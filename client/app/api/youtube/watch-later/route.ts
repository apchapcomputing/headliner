import { getServerSession } from 'next-auth/next';
import { NextResponse } from 'next/server';
import { YouTubeService } from '@/lib/services/youtube';

export async function GET() {
  try {
    const session = await getServerSession();

    if (!session?.accessToken) {
      return NextResponse.json({ error: 'Not authenticated' }, { status: 401 });
    }

    const youtubeService = new YouTubeService(session.accessToken);
    const watchLaterVideos = await youtubeService.getWatchLaterVideos();

    return NextResponse.json({ videos: watchLaterVideos });
  } catch (error) {
    console.error('Error in watch later videos route:', error);
    return NextResponse.json(
      { error: 'Failed to fetch watch later videos' },
      { status: 500 }
    );
  }
}
