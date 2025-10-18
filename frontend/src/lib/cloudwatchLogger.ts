import {
  CloudWatchLogsClient,
  CreateLogGroupCommand,
  CreateLogStreamCommand,
  PutLogEventsCommand,
} from "@aws-sdk/client-cloudwatch-logs";

const REGION = process.env.AWS_REGION ?? "us-east-1";
const LOG_GROUP = "/mavik/ssot/frontend-app/prod/access";

// Create CloudWatch Logs client
const cw = new CloudWatchLogsClient({ region: REGION });

// Generate log stream name by instance + date
function generateLogStreamName(): string {
  const date = new Date().toISOString().slice(0, 10); // YYYY-MM-DD
  const instanceId = process.env.INSTANCE_ID || 'frontend-app-prod';
  return `${instanceId}/${date}`;
}

// Ensure log group/stream exists at startup (idempotent)
export async function ensureLogsInfra() {
  const logStreamName = generateLogStreamName();
  
  try { 
    await cw.send(new CreateLogGroupCommand({ logGroupName: LOG_GROUP })); 
  } catch (error) {
    // Log group may already exist, ignore error
  }
  
  try { 
    await cw.send(new CreateLogStreamCommand({ 
      logGroupName: LOG_GROUP, 
      logStreamName: logStreamName 
    })); 
  } catch (error) {
    // Log stream may already exist, ignore error
  }
}

// Access log event interface
export interface AccessLogEvent {
  process_id: string;
  request_id: string;
  ts: number;                 // millisecond timestamp
  email: string;
  ip: string;
  route: string;              // e.g. "GET /api/graphql" or "DOWNLOAD /loans/csv"
  method: string;
  status: number;
  duration_ms: number;
  action: string;             // e.g. "GET_LOANS" / "DOWNLOAD_CSV"
  description?: string;
  user_agent?: string;
}

// Write to CloudWatch Logs only for "Fetch Data / Download CSV" actions
export async function logAccessIfNeeded(evt: AccessLogEvent) {
  const isTargetAction =
    evt.action === "GET_LOANS" ||
    evt.action === "DOWNLOAD_CSV" ||
    /\/csv(\?|$)/i.test(evt.route); // custom matching rule

  if (!isTargetAction) return;

  const logStreamName = generateLogStreamName();
  
  const logEvents = [{
    timestamp: Date.now(), // CloudWatch expects millisecond timestamp
    message: JSON.stringify({
      process_id: evt.process_id,
      request_id: evt.request_id,
      ts: evt.ts,
      email: evt.email,
      ip: evt.ip,
      route: evt.route,
      method: evt.method,
      status: evt.status,
      duration_ms: evt.duration_ms,
      action: evt.action,
      description: evt.description,
      user_agent: evt.user_agent,
    }),
  }];

  try {
    await cw.send(new PutLogEventsCommand({
      logGroupName: LOG_GROUP,
      logStreamName: logStreamName,
      logEvents,
    }));
  } catch (error) {
    console.error('Failed to send log to CloudWatch:', error);
    // Don't let logging failures affect business logic
  }
}

// Helper function to create access log events
export function createAccessLogEvent({
  request_id,
  email,
  ip,
  route,
  method,
  status,
  duration_ms,
  action,
  description,
  user_agent
}: Omit<AccessLogEvent, 'process_id' | 'ts'>): AccessLogEvent {
  return {
    process_id: process.env.INSTANCE_ID || 'frontend-app-prod',
    request_id,
    ts: Date.now(),
    email,
    ip,
    route,
    method,
    status,
    duration_ms,
    action,
    description,
    user_agent
  };
}