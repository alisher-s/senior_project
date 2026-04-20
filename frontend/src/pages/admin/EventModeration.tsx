import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Calendar, ChevronRight, Check } from 'lucide-react';
import { fetchAdminEvents, updateEventStatus } from '../api/admin';

type TabStatus = 'PENDING' | 'APPROVED' | 'REJECTED';

export default function AdminControlCenter() {
  const [activeTab, setActiveTab] = useState<TabStatus>('PENDING');
  const queryClient = useQueryClient();

  // 1. Fetch events based on the currently active tab
  const { data: events, isLoading } = useQuery({
    queryKey: ['adminEvents', activeTab],
    queryFn: () => fetchAdminEvents(activeTab),
  });

  // 2. Mutation to approve or reject an event
  const mutation = useMutation({
    mutationFn: updateEventStatus,
    onSuccess: () => {
      // This tells React Query to immediately refetch the lists,
      // making the event magically move to the correct tab!
      queryClient.invalidateQueries({ queryKey: ['adminEvents'] });
    },
  });

  const handleStatusChange = (eventId: string, newStatus: TabStatus) => {
    mutation.mutate({ eventId, status: newStatus });
  };

  return (
    <div className="min-h-screen bg-[#11131a] text-white p-4 font-sans">
      {/* Header */}
      <div className="flex items-center gap-4 mb-8 pt-4">
        <button className="p-2"><ChevronRight className="rotate-180" /></button>
        <h1 className="text-xl font-semibold text-[#E5B05C]">Admin Control Center</h1>
      </div>

      <div className="mb-6">
        <h2 className="text-lg font-bold mb-4">Event Moderation</h2>

        {/* Custom Tabs */}
        <div className="flex bg-[#1E202A] rounded-full p-1 border border-gray-700">
          {(['PENDING', 'APPROVED', 'REJECTED'] as TabStatus[]).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`flex-1 py-2 text-sm font-medium rounded-full transition-all flex justify-center items-center gap-2
                ${activeTab === tab 
                  ? 'bg-[#E5B05C] text-black' 
                  : 'text-gray-400 hover:text-white'}`}
            >
              {activeTab === tab && tab === 'PENDING' && <Check size={16} />}
              {activeTab === tab && tab === 'APPROVED' && <Check size={16} />}
              {tab.charAt(0) + tab.slice(1).toLowerCase()}
            </button>
          ))}
        </div>
      </div>

      {/* Event List */}
      <div className="space-y-4">
        {isLoading ? (
          <p className="text-gray-400 text-center py-4">Loading events...</p>
        ) : events?.length === 0 ? (
          <p className="text-gray-500 text-center py-4">No {activeTab.toLowerCase()} events.</p>
        ) : (
          events?.map((event) => (
            <div 
              key={event.id} 
              className={`bg-[#1E202A] border ${activeTab === 'APPROVED' ? 'border-[#E5B05C]' : 'border-gray-800'} rounded-2xl p-4 flex items-center justify-between`}
            >
              <div className="flex items-center gap-4">
                {/* Calendar Icon */}
                <div className="bg-[#2A2C38] p-3 rounded-xl">
                  <Calendar className="text-[#E5B05C]" size={24} />
                </div>
                
                {/* Event Info */}
                <div>
                  <h3 className="font-bold text-base">{event.title}</h3>
                  <p className="text-gray-400 text-xs mb-1">{event.date}</p>
                  
                  {/* Status Badge */}
                  <span className={`text-[10px] px-2 py-0.5 rounded-sm font-bold tracking-wide
                    ${event.status === 'APPROVED' ? 'bg-[#183321] text-[#4ade80]' : 
                      event.status === 'PENDING' ? 'bg-[#332415] text-[#fb923c]' : 
                      'bg-[#331515] text-[#f87171]'}`}
                  >
                    {event.status}
                  </span>
                </div>
              </div>

              {/* Action Buttons: If Pending, show Approve/Reject. Otherwise show arrow */}
              {activeTab === 'PENDING' ? (
                <div className="flex flex-col gap-2">
                  <button 
                    onClick={() => handleStatusChange(event.id, 'APPROVED')}
                    className="text-xs bg-green-600 hover:bg-green-500 text-white px-3 py-1 rounded"
                  >
                    Approve
                  </button>
                  <button 
                    onClick={() => handleStatusChange(event.id, 'REJECTED')}
                    className="text-xs bg-red-600 hover:bg-red-500 text-white px-3 py-1 rounded"
                  >
                    Reject
                  </button>
                </div>
              ) : (
                <button className="p-2 text-gray-400">
                  {activeTab === 'APPROVED' ? <Check className="text-[#E5B05C]" /> : <ChevronRight />}
                </button>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
