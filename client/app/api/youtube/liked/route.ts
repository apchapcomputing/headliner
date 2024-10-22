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
    const likedVideos = await youtubeService.getLikedVideos();

    return NextResponse.json({ videos: likedVideos });
  } catch (error) {
    console.error('Error in liked videos route:', error);
    return NextResponse.json(
      { error: 'Failed to fetch liked videos' },
      { status: 500 }
    );
  }
}
