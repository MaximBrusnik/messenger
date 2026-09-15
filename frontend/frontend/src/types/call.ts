export type CallType = "audio" | "video";

export type CallStatus =
  | "ringing"
  | "active"
  | "ended"
  | "missed"
  | "rejected"
  | "cancelled";

export interface CallSession {
  id: number;
  caller_id: number;
  callee_id: number;
  call_type: CallType;
  status: CallStatus;
  started_at_ms: number;
  accepted_at_ms?: number;
  ended_at_ms?: number;
  duration_ms: number;
  end_reason?: string;
}

export type CallPhase =
  | "idle"
  | "outgoing" // we are calling someone
  | "incoming" // someone is calling us
  | "active"; // media is connected

export interface IceServer {
  urls: string | string[];
  username?: string;
  credential?: string;
}

export interface IceConfig {
  ice_servers: IceServer[];
}

// WebSocket signaling messages (client <-> call-service).
export type SignalingMessage =
  | { type: "CALL_INVITE"; payload: { callee_id: number; call_type: CallType } }
  | { type: "CALL_ACCEPT"; payload: { call_id: number } }
  | { type: "CALL_REJECT"; payload: { call_id: number; reason?: string } }
  | { type: "CALL_END"; payload: { call_id: number; reason?: string } }
  | { type: "CALL_SDP"; payload: { call_id: number; data: string } }
  | { type: "CALL_ICE"; payload: { call_id: number; data: string } };

export interface SignalingEvent {
  type: string;
  payload: Record<string, unknown>;
}