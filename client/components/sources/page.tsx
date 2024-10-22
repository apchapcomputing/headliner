'use client';
import { useState } from 'react';
import { signIn, signOut, useSession } from 'next-auth/react';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card';
import { BookOpen, FileText, Loader2 } from 'lucide-react';
import Youtube from '@/components/icons/youtube';
import { YouTubeTitle } from '@/types/YoutubeTitle';

export default function SourcesPage() {
  const { data: session, status } = useSession();
  const [loading, setLoading] = useState(false);
  const [videos, setVideos] = useState<YouTubeTitle[]>([]);
  const [error, setError] = useState<string | null>(null);

  const fetchYouTubeData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [likedResponse, watchLaterResponse] = await Promise.all([
        fetch('/api/youtube/liked'),
        fetch('/api/youtube/watch-later')
      ]);

      if (!likedResponse.ok || !watchLaterResponse.ok) {
        throw new Error('Failed to fetch YouTube data');
      }

      const likedData = await likedResponse.json();
      const watchLaterData = await watchLaterResponse.json();

      setVideos([...likedData.videos, ...watchLaterData.videos]);
    } catch (err) {
      setError('Failed to fetch YouTube data');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className='space-y-6'>
      <div className='grid gap-4 md:grid-cols-3'>
        <Card>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <Youtube />
              YouTube
            </CardTitle>
            <CardDescription>
              Connect your YouTube account to analyze liked and watch later
              videos
            </CardDescription>
          </CardHeader>
          <CardContent>
            {status === 'loading' ? (
              <Button disabled className='w-full'>
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                Loading...
              </Button>
            ) : session ? (
              <div className='space-y-4'>
                <Button
                  onClick={fetchYouTubeData}
                  className='w-full'
                  disabled={loading}>
                  {loading ? (
                    <>
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                      Fetching...
                    </>
                  ) : (
                    'Fetch Videos'
                  )}
                </Button>
                <Button
                  onClick={() => signOut()}
                  variant='outline'
                  className='w-full'>
                  Disconnect
                </Button>
              </div>
            ) : (
              <Button onClick={() => signIn('google')} className='w-full'>
                Connect YouTube
              </Button>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <BookOpen className='h-5 w-5' />
              Substack
            </CardTitle>
            <CardDescription>
              Import headlines from your favorite Substack newsletters
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button className='w-full' variant='outline'>
              Connect Substack
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <FileText className='h-5 w-5' />
              Medium
            </CardTitle>
            <CardDescription>
              Analyze headlines from your Medium reading history
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button className='w-full' variant='outline'>
              Connect Medium
            </Button>
          </CardContent>
        </Card>
      </div>

      {error && (
        <div className='bg-red-50 text-red-600 p-4 rounded-lg'>{error}</div>
      )}

      {videos.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Fetched Video Titles</CardTitle>
            <CardDescription>
              {videos.length} videos found from your YouTube account
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ul className='space-y-2'>
              {videos.map((video) => (
                <li
                  key={video.id}
                  className='p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors'>
                  <div className='font-medium'>{video.title}</div>
                  <div className='text-sm text-gray-500'>
                    {video.channelTitle} •{' '}
                    {new Date(video.publishedAt).toLocaleDateString()}
                  </div>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
