import { redirect } from '@sveltejs/kit';

export const load = () => {
	// eslint-disable-next-line @typescript-eslint/only-throw-error -- SvelteKit redirects throw framework responses.
	throw redirect(307, '/device');
};
