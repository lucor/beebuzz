export interface VersionInfo {
	version: string;
	commit: string | null;
	dirty?: boolean;
}

export interface VersionDisplay {
	primary: string;
	secondary: string | null;
	badge: string | null;
	isDev: boolean;
}

export function formatVersionDisplay(info: VersionInfo): VersionDisplay {
	const isDev = info.version === 'dev';
	if (isDev) {
		const suffix = info.commit && info.commit !== 'dev' ? `@${info.commit}` : '';
		return {
			primary: `dev${suffix}`,
			secondary: null,
			badge: info.dirty ? 'dirty' : null,
			isDev: true
		};
	}
	return {
		primary: info.version,
		secondary: info.commit && info.commit !== 'dev' ? `commit ${info.commit.slice(0, 7)}` : null,
		badge: info.dirty ? 'dirty' : null,
		isDev: false
	};
}
