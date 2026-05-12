// ─── Pagination / Lists ───────────────────────────────────────────────────────────────

export interface Paginated<T> {
	data: T[];
	total: number;
	page: number;
	limit: number;
}

export interface ListResponse<T> {
	data: T[];
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export interface TokenPair {
	accessToken: string;
	refreshToken: string;
	expiresAt: string;
}

// ─── Host ─────────────────────────────────────────────────────────────────────

export interface Host {
	id: string;
	name: string;
	description: string;
	ip: string;
	imageId?: string;
	kernel: string;
	init: string;
	kernelArgs: string;
	isEnabled: boolean;
	useAad: boolean;
	useWol: boolean;
	lastContact?: string;
	deployedAt?: string;
	createdAt: string;
	updatedAt: string;
	macs?: HostMAC[];
}

export interface HostMAC {
	id: string;
	hostId: string;
	mac: string;
	description: string;
	isPrimary: boolean;
	isIgnored: boolean;
	createdAt: string;
}

export interface PendingMAC {
	id: string;
	mac: string;
	hostId?: string;
	firstSeen: string;
	lastSeen: string;
}

// ─── Image ────────────────────────────────────────────────────────────────────

export interface Image {
	id: string;
	name: string;
	description: string;
	path: string;
	osTypeId?: string;
	imageTypeId?: string;
	storageGroupId?: string;
	isEnabled: boolean;
	toReplicate: boolean;
	sizeBytes: number;
	partitions?: unknown;
	createdAt: string;
	createdBy: string;
	updatedAt: string;
}

// ─── Task ─────────────────────────────────────────────────────────────────────

export interface Task {
	id: string;
	name: string;
	type: string;
	state: string;
	hostId: string;
	imageId?: string;
	storageNodeId?: string;
	storageGroupId?: string;
	isGroup: boolean;
	isForced: boolean;
	isShutdown: boolean;
	percentComplete: number;
	bitsPerMinute: number;
	bytesTransferred: number;
	scheduledAt?: string;
	startedAt?: string;
	completedAt?: string;
	createdAt: string;
	createdBy: string;
	updatedAt: string;
}

// ─── Group ────────────────────────────────────────────────────────────────────

export interface Group {
	id: string;
	name: string;
	description: string;
	createdAt: string;
	createdBy: string;
	updatedAt: string;
	hostCount?: number;
}

export interface GroupMember {
	id: string;
	groupId: string;
	hostId: string;
}

// ─── Snapin ───────────────────────────────────────────────────────────────────

export interface Snapin {
	id: string;
	name: string;
	description: string;
	runOrder: number;
	timeout: number;
	fileName: string;
	filePath: string;
	command: string;
	arguments: string;
	runWith: string;
	hash: string;
	sizeBytes: number;
	isEnabled: boolean;
	toReplicate: boolean;
	createdAt: string;
	createdBy: string;
	updatedAt: string;
}

// ─── Storage ──────────────────────────────────────────────────────────────────

export interface StorageGroup {
	id: string;
	name: string;
	description: string;
	createdAt: string;
	updatedAt: string;
}

export interface StorageNode {
	id: string;
	name: string;
	description: string;
	storageGroupId: string;
	ip: string;
	path: string;
	hostname: string;
	rootPath: string;
	isEnabled: boolean;
	isMaster: boolean;
	isOnline: boolean;
	maxClients: number;
	bandwidthMbps: number;
	sshUser: string;
	webRoot: string;
	createdAt: string;
	updatedAt: string;
}

// ─── User ─────────────────────────────────────────────────────────────────────

export interface User {
	id: string;
	username: string;
	role: string;
	email: string;
	isActive: boolean;
	createdAt: string;
	createdBy: string;
	updatedAt: string;
	lastLoginAt?: string;
	lastLogin?: string;
}

// ─── Settings ─────────────────────────────────────────────────────────────────

export interface GlobalSetting {
	key: string;
	value: string;
	category: string;
	description: string;
}

// ─── Reports ──────────────────────────────────────────────────────────────────

export interface ImagingLog {
	id: string;
	hostId: string;
	taskId: string;
	taskType: string;
	type: string;
	state: string;
	imageId?: string;
	sizeBytes: number;
	duration: number;
	durationSeconds?: number;
	createdAt: string;
}

// ─── Inventory ────────────────────────────────────────────────────────────────

export interface Inventory {
	id: string;
	hostId: string;
	cpuModel: string;
	cpuCores: number;
	cpuFreqMhz: number;
	ramMib: number;
	hdModel: string;
	hdSizeGb: number;
	manufacturer: string;
	product: string;
	serial: string;
	uuid: string;
	biosVersion: string;
	primaryMac: string;
	osName: string;
	osVersion: string;
	createdAt: string;
	updatedAt: string;
}

// ─── Agent Logs ───────────────────────────────────────────────────────────────

export interface AgentLog {
	id: string;
	taskId: string;
	hostId: string;
	loggedAt: string;
	level: string;
	message: string;
	attrs?: Record<string, unknown>;
	createdAt: string;
}

export interface AgentLogEntry {
	time: string;
	level: string;
	msg: string;
	attrs?: Record<string, unknown>;
}

/** Payload carried by the `agent.log` WebSocket event. */
export interface AgentLogEvent {
	taskId: string;
	hostId: string;
	entries: AgentLogEntry[];
}
