import { google } from 'googleapis';
import { YouTubeTitle } from '@/types/YoutubeTitle';

export class YouTubeService {
  private youtube;

  constructor(accessToken: string) {
    const oauth2Client = new google.auth.OAuth2();
    oauth2Client.setCredentials({ access_token: accessToken });

    this.youtube = google.youtube({
      version: 'v3',
      auth: oauth2Client
    });
  }

  private async getPlaylistItems(playlistId: string): Promise<YouTubeTitle[]> {
    try {
      const response = await this.youtube.playlistItems.list({
        part: ['snippet'],
        playlistId: playlistId,
        maxResults: 50
      });

      return (
        response.data.items?.map((item) => ({
          id: item.id!,
          title: item.snippet!.title!,
          videoId: item.snippet!.resourceId!.videoId!,
          publishedAt: item.snippet!.publishedAt!,
          channelTitle: item.snippet!.channelTitle!
        })) || []
      );
    } catch (error) {
      console.error('Error fetching playlist items:', error);
      throw new Error('Failed to fetch playlist items');
    }
  }

  async getLikedVideos(): Promise<YouTubeTitle[]> {
    try {
      // First, get the liked videos playlist ID
      const response = await this.youtube.channels.list({
        part: ['contentDetails']
      });

      console.log(response);

      const likedPlaylistId =
        response.data.items?.[0].contentDetails?.relatedPlaylists?.likes;

      if (!likedPlaylistId) {
        throw new Error('Could not find liked videos playlist');
      }

      return this.getPlaylistItems(likedPlaylistId);
    } catch (error) {
      console.error('Error fetching liked videos:', error);
      throw new Error('Failed to fetch liked videos');
    }
  }

  async getWatchLaterVideos(): Promise<YouTubeTitle[]> {
    try {
      const response = await this.youtube.channels.list({
        part: ['contentDetails']
      });

      const watchLaterPlaylistId =
        response.data.items?.[0].contentDetails?.relatedPlaylists?.watchLater;

      if (!watchLaterPlaylistId) {
        throw new Error('Could not find watch later playlist');
      }

      return this.getPlaylistItems(watchLaterPlaylistId);
    } catch (error) {
      console.error('Error fetching watch later videos:', error);
      throw new Error('Failed to fetch watch later videos');
    }
  }
}
