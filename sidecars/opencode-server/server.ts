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

// Environment configuration
const PORT = parseInt(process.env.PORT || "3003", 10);
const WORKSPACE_DIR = process.env.WORKSPACE_DIR || "/workspace";
const LOG_LEVEL = process.env.LOG_LEVEL || "info";
const SESSION_TIMEOUT = parseInt(process.env.SESSION_TIMEOUT || "3600", 10);
const MAX_CONCURRENT_SESSIONS = parseInt(process.env.MAX_CONCURRENT_SESSIONS || "5", 10);

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

    // Create session state
    const now = new Date().toISOString();
    const session: SessionState = {
      sessionId: body.session_id,
      prompt: body.prompt,
      modelConfig: body.model_config,
      systemPrompt: body.system_prompt,
      status: "running",
      createdAt: now,
      lastActivity: now,
      progress: 0,
      controller: new AbortController()
    };

    sessions.set(body.session_id, session);

    log("info", "Session created", { 
      sessionId: body.session_id, 
      provider: body.model_config.provider 
    });

    // Start async session execution (placeholder for OpenCode integration)
    executeSession(session).catch(err => {
      log("error", "Session execution failed", { 
        sessionId: session.sessionId, 
        error: err.message 
      });
      session.status = "failed";
    });

    return Response.json(
      {
        session_id: body.session_id,
        status: "running",
        created_at: now
      },
      { status: 201 }
    );
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

// Placeholder session execution (will integrate with OpenCode in Phase 2.3)
async function executeSession(session: SessionState): Promise<void> {
  log("info", "Starting session execution", { sessionId: session.sessionId });
  
  // Simulate work for MVP (will be replaced with real OpenCode execution)
  await new Promise(resolve => setTimeout(resolve, 100));
  
  session.status = "completed";
  session.lastActivity = new Date().toISOString();
  session.progress = 100;
  
  log("info", "Session completed", { sessionId: session.sessionId });
}

// SSE stream endpoint
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

  log("info", "SSE stream started", { sessionId });

  // Create SSE stream using ReadableStream
  let eventId = 0;
  const stream = new ReadableStream({
    async start(controller) {
      const encoder = new TextEncoder();
      
      // Helper to send SSE event
      const sendEvent = (eventType: string, data: object) => {
        eventId++;
        const message = `event: ${eventType}\nid: ${eventId}\ndata: ${JSON.stringify(data)}\n\n`;
        controller.enqueue(encoder.encode(message));
      };

      try {
        // Send initial status event
        sendEvent("status", {
          status: session.status,
          timestamp: new Date().toISOString()
        });

        // Simulate streaming output (MVP - will integrate with OpenCode in Phase 2.3)
        sendEvent("output", {
          type: "stdout",
          text: `Starting task: ${session.prompt}`,
          timestamp: new Date().toISOString()
        });

        await new Promise(resolve => setTimeout(resolve, 500));

        sendEvent("tool_call", {
          tool: "read",
          args: { file: "/workspace/README.md" },
          timestamp: new Date().toISOString()
        });

        await new Promise(resolve => setTimeout(resolve, 300));

        sendEvent("tool_result", {
          tool: "read",
          result: { content: "File content..." },
          timestamp: new Date().toISOString()
        });

        await new Promise(resolve => setTimeout(resolve, 500));

        sendEvent("output", {
          type: "stdout",
          text: "Task execution completed successfully",
          timestamp: new Date().toISOString()
        });

        sendEvent("complete", {
          final_message: "Task completed",
          files_modified: [],
          timestamp: new Date().toISOString()
        });

        // Heartbeat interval (send every 30 seconds to keep connection alive)
        const heartbeatInterval = setInterval(() => {
          if (session.status === "completed" || session.status === "failed" || session.status === "cancelled") {
            clearInterval(heartbeatInterval);
            controller.close();
          } else {
            sendEvent("heartbeat", {});
          }
        }, 30000);

        // Close after completion events
        await new Promise(resolve => setTimeout(resolve, 100));
        clearInterval(heartbeatInterval);
        controller.close();
        
      } catch (error) {
        log("error", "SSE stream error", { 
          sessionId, 
          error: error instanceof Error ? error.message : String(error) 
        });
        sendEvent("error", {
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

// Cancel session endpoint
function handleCancelSession(sessionId: string): Response {
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

  // Cancel the session
  session.controller?.abort();
  session.status = "cancelled";
  session.lastActivity = new Date().toISOString();

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

log("info", "OpenCode Server started", { 
  port: PORT, 
  workspace: WORKSPACE_DIR,
  logLevel: LOG_LEVEL
});
