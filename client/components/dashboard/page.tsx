// components/dashboard/page.tsx
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription
} from '@/components/ui/card';
import { ThumbsUp, Award, Zap, History } from 'lucide-react';

export default function DashboardPage() {
  return (
    <div className='space-y-6'>
      {/* Top performing headline examples */}
      <Card>
        <CardHeader>
          <CardTitle>Top Performing Headlines</CardTitle>
          <CardDescription>Based on source content engagement</CardDescription>
        </CardHeader>
        <CardContent>
          <div className='space-y-4'>
            <div className='flex items-start gap-2'>
              <ThumbsUp className='w-5 h-5 text-green-500 mt-1' />
              <div>
                <p className='font-medium'>
                  10 Unexpected Ways to Boost Your Productivity
                </p>
                <p className='text-sm text-muted-foreground'>
                  98% completion rate from YouTube
                </p>
              </div>
            </div>
            <div className='flex items-start gap-2'>
              <ThumbsUp className='w-5 h-5 text-green-500 mt-1' />
              <div>
                <p className='font-medium'>
                  The Hidden Truth About Work-Life Balance
                </p>
                <p className='text-sm text-muted-foreground'>
                  2.5k claps on Medium
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
        {/* Content Source Overview */}
        <Card>
          <CardHeader>
            <CardTitle>Content Sources</CardTitle>
            <CardDescription>
              Connected platforms and content volume
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className='space-y-4'>
              <div className='flex justify-between items-center'>
                <span>YouTube</span>
                <div className='text-right'>
                  <div className='text-green-600'>Connected</div>
                  <div className='text-sm text-muted-foreground'>
                    142 saved videos
                  </div>
                </div>
              </div>
              <div className='flex justify-between items-center'>
                <span>Substack</span>
                <div className='text-right'>
                  <div className='text-gray-400'>Not Connected</div>
                  <div className='text-sm text-muted-foreground'>-</div>
                </div>
              </div>
              <div className='flex justify-between items-center'>
                <span>Medium</span>
                <div className='text-right'>
                  <div className='text-gray-400'>Not Connected</div>
                  <div className='text-sm text-muted-foreground'>-</div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Pattern Analysis */}
        <Card>
          <CardHeader>
            <CardTitle>Common Patterns</CardTitle>
            <CardDescription>Successful headline structures</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='space-y-4'>
              <div className='flex items-start gap-2'>
                <Award className='w-5 h-5 text-blue-500 mt-1' />
                <div>
                  <p className='font-medium'>How-to Format</p>
                  <p className='text-sm text-muted-foreground'>
                    32% of top performing content
                  </p>
                </div>
              </div>
              <div className='flex items-start gap-2'>
                <Award className='w-5 h-5 text-blue-500 mt-1' />
                <div>
                  <p className='font-medium'>Number Lists</p>
                  <p className='text-sm text-muted-foreground'>
                    28% of top performing content
                  </p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Recent Activity */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>Latest generations and updates</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='space-y-4'>
              <div className='flex items-start gap-2'>
                <History className='w-5 h-5 text-purple-500 mt-1' />
                <div>
                  <p className='text-sm font-medium'>
                    Generated 5 headlines for "productivity tips"
                  </p>
                  <p className='text-xs text-muted-foreground'>2 minutes ago</p>
                </div>
              </div>
              <div className='flex items-start gap-2'>
                <Zap className='w-5 h-5 text-yellow-500 mt-1' />
                <div>
                  <p className='text-sm font-medium'>
                    Connected YouTube account
                  </p>
                  <p className='text-xs text-muted-foreground'>1 hour ago</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
