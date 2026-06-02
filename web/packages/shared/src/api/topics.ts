// Topics API endpoints.
import { api } from './client';

export interface Topic {
	id: string;
	name: string;
	description?: string;
	created_at: string;
	updated_at: string;
}

/**
 * Topics API namespace.
 */
export const topicsApi = {
	/**
	 * List all topics for current user.
	 */
	listTopics: async () => {
		const data = await api.get<{ data: Topic[] }>('/topics');
		return data.data || [];
	},

	/**
	 * Create new topic.
	 */
	createTopic: (name: string, description: string = '') =>
		api.post<Topic>('/topics', { name, description }),

	/**
	 * Update topic description.
	 */
	updateTopic: (topicId: string, description: string = '') =>
		api.patch<void>(`/topics/${topicId}`, { description }),

	/**
	 * Delete topic.
	 */
	deleteTopic: (topicId: string) => api.delete<void>(`/topics/${topicId}`)
};
