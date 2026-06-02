type LogData = Record<string, unknown>;

/** Formats a log message with optional structured data */
const formatLog = (message: string, data?: LogData): [string] | [string, LogData] => {
	if (data) {
		return [message, data];
	}
	return [message];
};

export const logger = {
	/** Logs debug messages. Only outputs when VITE_BEEBUZZ_DEBUG=true. */
	debug: (message: string, data?: LogData) => {
		if (import.meta.env.VITE_BEEBUZZ_DEBUG === true) console.debug(...formatLog(message, data));
	},
	/** Logs informational messages for important business events. */
	info: (message: string, data?: LogData) => console.info(...formatLog(message, data)),
	/** Logs warnings for recoverable issues or security events. */
	warn: (message: string, data?: LogData) => console.warn(...formatLog(message, data)),
	/** Logs errors for critical failures requiring attention. */
	error: (message: string, data?: LogData) => console.error(...formatLog(message, data))
};
