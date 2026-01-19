/**
 * OpenCode Server Sidecar
 * Provides REST API + SSE streaming for AI agent session management
 * 
 * Endpoints:
 * - GET /healthz - Liveness probe (200 OK)
 * - GET /ready - Readiness probe (200 OK / 503 Service Unavailable)
 * - POST /sessions - Create new session
 * - GET /sessions/:id/stream - SSE stream for session output
 * - DELETE /sessions/:id - Cancel session
 * - GET /sessions/:id/status - Get session status
 */

import { file, write as bunWrite } from "bun";
import { access, constants } from "fs/promises";
import { createOpencodeClient } from "@opencode-ai/sdk";
import type { Session as OpenCodeSession, Message, Part } from "@opencode-ai/sdk";

// Environment configuration
const PORT = parseInt(process.env.PORT || "3003", 10);
const WORKSPACE_DIR = process.env.WORKSPACE_DIR || "/workspace";
const LOG_LEVEL = process.env.LOG_LEVEL || "info";
const SESSION_TIMEOUT = parseInt(process.env.SESSION_TIMEOUT || "3600", 10);
const MAX_CONCURRENT_SESSIONS = parseInt(process.env.MAX_CONCURRENT_SESSIONS || "5", 10);
const BACKEND_API_URL = process.env.BACKEND_API_URL || "http://localhost:8090";

// Logger utility
const logLevels: Record<string, number> = { debug: 0, info: 1, warn: 2, error: 3 };
const currentLogLevel = logLevels[LOG_LEVEL] ?? 1;

function log(level: string, message: string, meta?: object) {
  if (logLevels[level] >= currentLogLevel) {
    const timestamp = new Date().toISOString();
    const logEntry = { timestamp, level, message, ...meta };
    console.log(JSON.stringify(logEntry));
  }
}

// Session state management (in-memory for MVP)
interface SessionState {
  sessionId: string;
  prompt: string;
  modelConfig: ModelConfig;
  systemPrompt?: string;
  status: "pending" | "running" | "waiting_input" | "completed" | "failed" | "cancelled";
  createdAt: string;
  lastActivity: string;
  progress: number;
  currentTool?: string;
  controller?: AbortController;
  opencodeSessionId?: string; // OpenCode SDK session ID
  error?: string; // Error message if failed
  lastEventId?: string; // Last processed OpenCode event ID for replay
  eventBuffer: Array<{ eventId: string; eventType: string; data: object }>; // Event buffer for reconnection
}

interface ModelConfig {
  provider: string;
  model: string;
  api_key: string;
  temperature: number;
  max_tokens: number;
  enabled_tools: string[];
  model_version?: string;
  api_endpoint?: string;
}

const sessions = new Map<string, SessionState>();

let opencodeClient: ReturnType<typeof createOpencodeClient> | null = null;

async function getOpencodeClient() {
  if (!opencodeClient) {
    opencodeClient = createOpencodeClient({
      baseUrl: process.env.OPENCODE_SERVER_URL || "http://localhost:3000"
    });
  }
  return opencodeClient;
}

// Health check endpoint
function handleHealthz(): Response {
  return Response.json({ status: "ok" });
}

// Readiness check endpoint
async function handleReady(): Promise<Response> {
  try {
    // Check if workspace directory is accessible
    await access(WORKSPACE_DIR, constants.R_OK | constants.W_OK);
    
    log("debug", "Readiness check passed", { workspace: WORKSPACE_DIR });
    return Response.json({ status: "ready" });
  } catch (error) {
    log("error", "Readiness check failed", { 
      workspace: WORKSPACE_DIR, 
      error: error instanceof Error ? error.message : String(error) 
    });
    return Response.json(
      { 
        status: "not ready", 
        error: "workspace not accessible",
        timestamp: new Date().toISOString()
      },
      { status: 503 }
    );
  }
}

// Create session endpoint
async function handleCreateSession(req: Request): Promise<Response> {
  try {
    const body = await req.json();
    
    // Validate required fields
    if (!body.session_id || typeof body.session_id !== "string") {
      return Response.json(
        { 
          error: "Invalid request: missing required field 'session_id'",
          timestamp: new Date().toISOString()
        },
        { status: 400 }
      );
    }
    
    if (!body.prompt || typeof body.prompt !== "string") {
      return Response.json(
        { 
          error: "Invalid request: missing required field 'prompt'",
          timestamp: new Date().toISOString()
        },
        { status: 400 }
      );
    }

    if (!body.model_config || typeof body.model_config !== "object") {
      return Response.json(
        { 
          error: "Invalid request: missing required field 'model_config'",
          timestamp: new Date().toISOString()
        },
        { status: 400 }
      );
    }

    // Check if session already exists
    if (sessions.has(body.session_id)) {
      return Response.json(
        { 
          error: `Session with ID ${body.session_id} already exists`,
          timestamp: new Date().toISOString()
        },
        { status: 409 }
      );
    }

    // Check concurrent session limit
    const runningSessions = Array.from(sessions.values()).filter(
      s => s.status === "running" || s.status === "pending"
    );
    if (runningSessions.length >= MAX_CONCURRENT_SESSIONS) {
      return Response.json(
        { 
          error: `Maximum concurrent sessions limit reached (${MAX_CONCURRENT_SESSIONS})`,
          timestamp: new Date().toISOString()
        },
        { status: 503 }
      );
    }

    const now = new Date().toISOString();
    const session: SessionState = {
      sessionId: body.session_id,
      prompt: body.prompt,
      modelConfig: body.model_config,
      systemPrompt: body.system_prompt,
      status: "pending",
      createdAt: now,
      lastActivity: now,
      progress: 0,
      controller: new AbortController(),
      eventBuffer: []
    };

    sessions.set(body.session_id, session);

    log("info", "Session created", { 
      sessionId: body.session_id, 
      provider: body.model_config.provider 
    });

    // Create OpenCode session SYNCHRONOUSLY to get remote_session_id
    try {
      const client = await getOpencodeClient();
      
      const createResult = await client.session.create({
        body: {
          title: `Session ${session.sessionId}`,
          workingDirectory: WORKSPACE_DIR
        }
      });
      
      if (!createResult.data) {
        throw new Error("Failed to create OpenCode session");
      }
      
      session.opencodeSessionId = createResult.data.id;
      session.status = "running";
      session.lastActivity = new Date().toISOString();
      
      log("info", "OpenCode session created", { 
        sessionId: session.sessionId,
        opencodeSessionId: session.opencodeSessionId
      });

      // Start async execution in background
      executeSessionAsync(session).catch(err => {
        log("error", "Session execution failed", { 
          sessionId: session.sessionId, 
          error: err.message 
        });
        session.status = "failed";
        session.error = err.message;
      });

      return Response.json(
        {
          session_id: body.session_id,
          remote_session_id: session.opencodeSessionId,
          status: "running",
          created_at: now
        },
        { status: 201 }
      );
    } catch (error) {
      session.status = "failed";
      session.error = error instanceof Error ? error.message : String(error);
      
      log("error", "Failed to create OpenCode session", { 
        sessionId: body.session_id,
        error: session.error
      });
      
      return Response.json(
        { 
          error: "Failed to create OpenCode session",
          details: session.error,
          timestamp: new Date().toISOString()
        },
        { status: 500 }
      );
    }
  } catch (error) {
    log("error", "Failed to create session", { 
      error: error instanceof Error ? error.message : String(error) 
    });
    return Response.json(
      { 
        error: "Failed to initialize OpenCode session",
        details: error instanceof Error ? error.message : String(error),
        timestamp: new Date().toISOString()
      },
      { status: 500 }
    );
  }
}

async function executeSessionAsync(session: SessionState): Promise<void> {
  log("info", "Starting OpenCode prompt execution", { 
    sessionId: session.sessionId,
    opencodeSessionId: session.opencodeSessionId
  });
  
  const timeoutId = setTimeout(() => {
    if (session.status === "running" || session.status === "pending") {
      session.controller?.abort();
      session.status = "failed";
      session.error = `Session timeout after ${SESSION_TIMEOUT} seconds`;
      session.lastActivity = new Date().toISOString();
      log("warn", "Session timeout", { 
        sessionId: session.sessionId,
        timeout: SESSION_TIMEOUT 
      });
    }
  }, SESSION_TIMEOUT * 1000);
  
  try {
    const client = await getOpencodeClient();
    
    if (!session.opencodeSessionId) {
      throw new Error("Missing remote session ID");
    }
    
    const sendResult = await client.session.prompt({
      path: { id: session.opencodeSessionId },
      body: {
        model: {
          providerID: session.modelConfig.provider,
          modelID: session.modelConfig.model,
          apiKey: session.modelConfig.api_key,
          temperature: session.modelConfig.temperature,
          maxTokens: session.modelConfig.max_tokens
        },
        parts: [
          ...(session.systemPrompt ? [{ type: "text" as const, text: session.systemPrompt }] : []),
          { type: "text" as const, text: session.prompt }
        ]
      }
    });
    
    if (session.controller?.signal.aborted) {
      session.status = "cancelled";
      log("info", "Session cancelled during execution", { sessionId: session.sessionId });
      clearTimeout(timeoutId);
      return;
    }
    
    session.status = "completed";
    session.lastActivity = new Date().toISOString();
    session.progress = 100;
    clearTimeout(timeoutId);
    
    log("info", "OpenCode session completed", { 
      sessionId: session.sessionId,
      opencodeSessionId: session.opencodeSessionId
    });
    
  } catch (error) {
    clearTimeout(timeoutId);
    
    if (session.controller?.signal.aborted) {
      session.status = "cancelled";
      log("info", "Session cancelled during execution", { sessionId: session.sessionId });
    } else {
      session.status = "failed";
      session.error = error instanceof Error ? error.message : String(error);
      session.lastActivity = new Date().toISOString();
      
      log("error", "OpenCode prompt execution failed", {
        sessionId: session.sessionId,
        opencodeSessionId: session.opencodeSessionId,
        error: session.error
      });
    }
  }
}

function handleSessionStream(sessionId: string, req: Request): Response {
  const session = sessions.get(sessionId);
  
  if (!session) {
    return Response.json(
      { 
        error: "Session not found",
        timestamp: new Date().toISOString()
      },
      { status: 404 }
    );
  }

  const lastEventId = req.headers.get("Last-Event-ID");
  log("info", "SSE stream started", { sessionId, lastEventId });

  const stream = new ReadableStream({
    async start(controller) {
      const encoder = new TextEncoder();
      
      const sendEvent = (eventId: string, eventType: string, data: object) => {
        const message = `event: ${eventType}\nid: ${eventId}\ndata: ${JSON.stringify(data)}\n\n`;
        controller.enqueue(encoder.encode(message));
        
        session.eventBuffer.push({ eventId, eventType, data });
        if (session.eventBuffer.length > 100) {
          session.eventBuffer.shift();
        }
        session.lastEventId = eventId;
      };

      try {
        if (lastEventId) {
          const replayIndex = session.eventBuffer.findIndex(e => e.eventId === lastEventId);
          if (replayIndex !== -1) {
            const eventsToReplay = session.eventBuffer.slice(replayIndex + 1);
            for (const event of eventsToReplay) {
              controller.enqueue(encoder.encode(
                `event: ${event.eventType}\nid: ${event.eventId}\ndata: ${JSON.stringify(event.data)}\n\n`
              ));
            }
            log("info", "Replayed events from buffer", { 
              sessionId, 
              count: eventsToReplay.length 
            });
          }
        }

        const initialEventId = `${session.opencodeSessionId || sessionId}-init-${Date.now()}`;
        sendEvent(initialEventId, "status", {
          status: session.status,
          timestamp: new Date().toISOString()
        });

        if (!session.opencodeSessionId) {
          const errorEventId = `${sessionId}-error-${Date.now()}`;
          sendEvent(errorEventId, "error", {
            error: "OpenCode session not initialized",
            fatal: true,
            timestamp: new Date().toISOString()
          });
          controller.close();
          return;
        }

        const client = await getOpencodeClient();
        const eventsResult = await client.event.subscribe();
        
        if (!eventsResult.data) {
          throw new Error("Failed to subscribe to OpenCode events");
        }

        const heartbeatInterval = setInterval(() => {
          if (session.status === "completed" || session.status === "failed" || session.status === "cancelled") {
            clearInterval(heartbeatInterval);
          } else {
            const heartbeatEventId = `${session.opencodeSessionId}-heartbeat-${Date.now()}`;
            sendEvent(heartbeatEventId, "heartbeat", {});
          }
        }, 30000);

        for await (const event of eventsResult.data.stream) {
          if (session.controller?.signal.aborted) {
            clearInterval(heartbeatInterval);
            const cancelEventId = event.id || `${session.opencodeSessionId}-cancel-${Date.now()}`;
            sendEvent(cancelEventId, "status", {
              status: "cancelled",
              timestamp: new Date().toISOString()
            });
            break;
          }

          const opencodeEventId = event.id || `${session.opencodeSessionId}-${event.type}-${Date.now()}`;

          if (event.type === "tool_call") {
            sendEvent(opencodeEventId, "tool_call", {
              tool: event.properties.tool,
              args: event.properties.args,
              timestamp: new Date().toISOString()
            });
            session.currentTool = event.properties.tool;
          } else if (event.type === "tool_result") {
            sendEvent(opencodeEventId, "tool_result", {
              tool: event.properties.tool,
              result: event.properties.result,
              timestamp: new Date().toISOString()
            });
            session.currentTool = undefined;
          } else if (event.type === "output") {
            sendEvent(opencodeEventId, "output", {
              type: event.properties.stream || "stdout",
              text: event.properties.text,
              timestamp: new Date().toISOString()
            });
          } else if (event.type === "progress") {
            session.progress = event.properties.percent || 0;
            sendEvent(opencodeEventId, "status", {
              status: session.status,
              progress: session.progress,
              timestamp: new Date().toISOString()
            });
          }

          if (session.status === "completed" || session.status === "failed") {
            clearInterval(heartbeatInterval);
            const completeEventId = event.id || `${session.opencodeSessionId}-complete-${Date.now()}`;
            sendEvent(completeEventId, "complete", {
              final_message: event.properties.message || "Task completed",
              files_modified: event.properties.files || [],
              timestamp: new Date().toISOString()
            });
            break;
          }
        }

        clearInterval(heartbeatInterval);
        controller.close();
        
      } catch (error) {
        log("error", "SSE stream error", { 
          sessionId, 
          error: error instanceof Error ? error.message : String(error) 
        });
        const errorEventId = `${session.opencodeSessionId || sessionId}-error-${Date.now()}`;
        sendEvent(errorEventId, "error", {
          error: error instanceof Error ? error.message : String(error),
          fatal: true,
          timestamp: new Date().toISOString()
        });
        controller.close();
      }
    }
  });

  return new Response(stream, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      "Connection": "keep-alive",
      "X-Accel-Buffering": "no"
    }
  });
}

async function handleCancelSession(sessionId: string): Promise<Response> {
  const session = sessions.get(sessionId);
  
  if (!session) {
    return Response.json(
      { 
        error: "Session not found",
        timestamp: new Date().toISOString()
      },
      { status: 404 }
    );
  }

  session.controller?.abort();
  session.status = "cancelled";
  session.lastActivity = new Date().toISOString();

  if (session.opencodeSessionId) {
    try {
      const client = await getOpencodeClient();
      await client.session.abort({
        path: { id: session.opencodeSessionId }
      });
      log("info", "OpenCode session aborted", { 
        sessionId, 
        opencodeSessionId: session.opencodeSessionId 
      });
    } catch (error) {
      log("error", "Failed to abort OpenCode session", {
        sessionId,
        opencodeSessionId: session.opencodeSessionId,
        error: error instanceof Error ? error.message : String(error)
      });
    }
  }

  log("info", "Session cancelled", { sessionId });

  return Response.json({
    session_id: sessionId,
    status: "cancelled",
    cancelled_at: new Date().toISOString()
  });
}

// Get session status endpoint
function handleSessionStatus(sessionId: string): Response {
  const session = sessions.get(sessionId);
  
  if (!session) {
    return Response.json(
      { 
        error: "Session not found",
        timestamp: new Date().toISOString()
      },
      { status: 404 }
    );
  }

  return Response.json({
    session_id: sessionId,
    status: session.status,
    created_at: session.createdAt,
    last_activity: session.lastActivity,
    progress: session.progress,
    current_tool: session.currentTool
  });
}

// Main request router
async function handleRequest(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const path = url.pathname;
  const method = req.method;

  log("debug", "Request received", { method, path });

  // Health checks
  if (method === "GET" && (path === "/healthz" || path === "/health")) {
    return handleHealthz();
  }

  if (method === "GET" && path === "/ready") {
    return handleReady();
  }

  // Session management
  if (method === "POST" && path === "/sessions") {
    return handleCreateSession(req);
  }

  // Session streaming
  const streamMatch = path.match(/^\/sessions\/([^/]+)\/stream$/);
  if (method === "GET" && streamMatch) {
    return handleSessionStream(streamMatch[1], req);
  }

  // Session cancellation
  const cancelMatch = path.match(/^\/sessions\/([^/]+)$/);
  if (method === "DELETE" && cancelMatch) {
    return handleCancelSession(cancelMatch[1]);
  }

  // Session status
  const statusMatch = path.match(/^\/sessions\/([^/]+)\/status$/);
  if (method === "GET" && statusMatch) {
    return handleSessionStatus(statusMatch[1]);
  }

  // 404 Not Found
  return Response.json(
    { 
      error: "Not found",
      path,
      timestamp: new Date().toISOString()
    },
    { status: 404 }
  );
}

// Graceful shutdown handler
function setupGracefulShutdown(server: any) {
  const shutdown = async (signal: string) => {
    log("info", `Received ${signal}, shutting down gracefully`);
    
    // Cancel all running sessions
    for (const session of sessions.values()) {
      if (session.status === "running" || session.status === "pending") {
        session.controller?.abort();
        session.status = "cancelled";
      }
    }
    
    server.stop();
    process.exit(0);
  };

  process.on("SIGTERM", () => shutdown("SIGTERM"));
  process.on("SIGINT", () => shutdown("SIGINT"));
}

async function recoverActiveSessions() {
  log("info", "Starting session recovery from backend database");
  
  try {
    const response = await fetch(`${BACKEND_API_URL}/api/sessions/active`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json"
      }
    });

    if (!response.ok) {
      log("warn", "Failed to fetch active sessions", { 
        status: response.status,
        statusText: response.statusText
      });
      return;
    }

    const data = await response.json();
    const activeSessions = data.sessions || [];

    log("info", "Active sessions found in database", { count: activeSessions.length });

    for (const dbSession of activeSessions) {
      try {
        if (!dbSession.remote_session_id) {
          log("warn", "Session missing remote_session_id, marking as failed", {
            sessionId: dbSession.id
          });
          await markSessionFailed(dbSession.id, "Missing OpenCode session ID");
          continue;
        }

        const client = await getOpencodeClient();
        const sessionStatus = await client.session.get({
          path: { id: dbSession.remote_session_id }
        });

        if (sessionStatus.data) {
          log("info", "Recovered session from OpenCode", {
            sessionId: dbSession.id,
            remoteSessionId: dbSession.remote_session_id,
            opencodeStatus: sessionStatus.data.status
          });

          const recoveredSession: SessionState = {
            sessionId: dbSession.id,
            prompt: dbSession.prompt || "",
            modelConfig: {
              provider: dbSession.model_config?.provider || "openai",
              model: dbSession.model_config?.model || "gpt-4o-mini",
              api_key: "",
              temperature: dbSession.model_config?.temperature || 0.7,
              max_tokens: dbSession.model_config?.max_tokens || 4096,
              enabled_tools: dbSession.model_config?.enabled_tools || []
            },
            status: mapOpencodeStatus(sessionStatus.data.status),
            createdAt: dbSession.created_at,
            lastActivity: new Date().toISOString(),
            progress: 0,
            controller: new AbortController(),
            opencodeSessionId: dbSession.remote_session_id,
            lastEventId: dbSession.last_event_id,
            eventBuffer: []
          };

          sessions.set(dbSession.id, recoveredSession);
        } else {
          log("warn", "OpenCode session not found, marking as failed", {
            sessionId: dbSession.id,
            remoteSessionId: dbSession.remote_session_id
          });
          await markSessionFailed(dbSession.id, "OpenCode session no longer exists");
        }
      } catch (error) {
        log("error", "Failed to recover session", {
          sessionId: dbSession.id,
          error: error instanceof Error ? error.message : String(error)
        });
        await markSessionFailed(dbSession.id, 
          error instanceof Error ? error.message : "Unknown recovery error");
      }
    }

    log("info", "Session recovery complete", { 
      recovered: sessions.size,
      total: activeSessions.length 
    });
  } catch (error) {
    log("error", "Session recovery failed", {
      error: error instanceof Error ? error.message : String(error)
    });
  }
}

function mapOpencodeStatus(opencodeStatus: string): SessionState["status"] {
  switch (opencodeStatus) {
    case "running":
      return "running";
    case "completed":
      return "completed";
    case "failed":
      return "failed";
    case "cancelled":
      return "cancelled";
    default:
      return "pending";
  }
}

async function markSessionFailed(sessionId: string, reason: string) {
  try {
    await fetch(`${BACKEND_API_URL}/api/sessions/${sessionId}/status`, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        status: "failed",
        error: reason
      })
    });
  } catch (error) {
    log("error", "Failed to update session status in backend", {
      sessionId,
      error: error instanceof Error ? error.message : String(error)
    });
  }
}

// Start server
const server = Bun.serve({
  port: PORT,
  fetch: handleRequest,
  error(error) {
    log("error", "Server error", { error: error.message });
    return Response.json(
      { 
        error: "Internal server error",
        timestamp: new Date().toISOString()
      },
      { status: 500 }
    );
  }
});

setupGracefulShutdown(server);

recoverActiveSessions().catch(err => {
  log("error", "Startup recovery failed", { error: err.message });
});

log("info", "OpenCode Server started", { 
  port: PORT, 
  workspace: WORKSPACE_DIR,
  logLevel: LOG_LEVEL,
  backendApiUrl: BACKEND_API_URL
});
