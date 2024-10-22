import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

import DashboardPage from '@/components/dashboard/page';
import SourcesPage from '@/components/sources/page';
import GeneratorPage from '@/components/generator/page';

export default function Home() {
  return (
    <div className='container mx-auto py-6'>
      <h1 className='text-3xl font-bold mb-6'>Headliner</h1>
      <Tabs defaultValue='dashboard' className='space-y-4'>
        <TabsList>
          <TabsTrigger value='dashboard'>Dashboard</TabsTrigger>
          <TabsTrigger value='sources'>Content Sources</TabsTrigger>
          <TabsTrigger value='generator'>Generate Headlines</TabsTrigger>
        </TabsList>
        <TabsContent value='dashboard'>
          <DashboardPage />
        </TabsContent>
        <TabsContent value='sources'>
          <SourcesPage />
        </TabsContent>
        <TabsContent value='generator'>
          <GeneratorPage />
        </TabsContent>
      </Tabs>
    </div>
  );
}
