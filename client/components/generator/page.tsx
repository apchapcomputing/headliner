// components/generator/page.tsx
'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Loader2 } from 'lucide-react';

export default function GeneratorPage() {
  const [topic, setTopic] = useState('');
  const [loading, setLoading] = useState(false);
  const [headlines, setHeadlines] = useState<string[]>([]);

  const generateHeadlines = async () => {
    setLoading(true);
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 2000));
    setHeadlines([
      '10 Unexpected Ways to Boost Your Productivity',
      'The Hidden Truth About Work-Life Balance',
      'Why Everything You Know About Success Is Wrong',
      '5 Game-Changing Habits of Successful People',
      'The Science Behind Peak Performance'
    ]);
    setLoading(false);
  };

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <CardTitle>Generate Headlines</CardTitle>
        </CardHeader>
        <CardContent>
          <div className='space-y-4'>
            <div className='space-y-2'>
              <Label htmlFor='topic'>Topic</Label>
              <Input
                id='topic'
                placeholder='Enter your topic...'
                value={topic}
                onChange={(e) => setTopic(e.target.value)}
              />
            </div>
            <Button
              onClick={generateHeadlines}
              disabled={loading || !topic}
              className='w-full'>
              {loading ? (
                <>
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  Generating...
                </>
              ) : (
                'Generate Headlines'
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      {headlines.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Generated Headlines</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className='space-y-2'>
              {headlines.map((headline, index) => (
                <li
                  key={index}
                  className='p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors'>
                  {headline}
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
