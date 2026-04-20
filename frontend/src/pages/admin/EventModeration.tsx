import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Calendar, ChevronLeft, ChevronRight, Check, X, Search } from 'lucide-react';
import { adminAPI } from '../../api/services';

// Define our types
type TabStatus = 'PENDING' | 'APPROVED' | 'REJECTED';

interface EventItem {
  id: string;
  title: string;
  date?: string; // or start_time / created_at depending on your Go struct
  status: TabStatus;
}

export default function EventModeration() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const[activeTab, setActiveTab] = useState<TabStatus>('PENDING');

  // 1. Fetch Events based on active tab
  const { data: response, isLoading } = useQuery({
    queryKey:['adminEvents', activeTab],
    queryFn: () => adminAPI.getEvents({ status: activeTab }),
  });

  // Extract events (Adjust '.events' if your Go backend returns the array directly in response.data)
  const events: EventItem[] = response?.data?.events || response?.data ||[];

  // 2. Mutation to Approve or Reject
  const moderateMutation = useMutation({
    mutationFn: ({ eventId, action }: { eventId: string, action: 'APPROVED' | 'REJECTED' }) => 
      // NOTE: Change '{ status: action }' if your ModerateEventRequest expects '{ action: "approve" }'
      adminAPI.moderateEvent(eventId, { status: action } as any), 
    onSuccess: () => {
      // Instantly refresh the lists so the event moves to the correct tab!
      queryClient.invalidateQueries({ queryKey: ['adminEvents'] });
    },
  });

  const handleModerate = (eventId: string, action: 'APPROVED' | 'REJECTED') => {
    moderateMutation.mutate({ eventId, action });
  };

  return (
    <div className="min-h-screen bg-[#11131a] text-white p-4 font-sans pb-10">
      
      {/* ─── Header ──────────────────────────────────────── */}
      <div className="flex items-center gap-4 mb-8 pt-2">
        <button onClick={() => navigate(-1)} className="p-2 -ml-2 text-white hover:text-gray-300">
          <ChevronLeft size={28} />
        </button>
        <h1 className="text-xl font-bold text-[#E5B05C]">Admin Control Center</h1>
      </div>

      {/* ─── Event Moderation Section ────────────────────── */}
      <div className="mb-8">
        <h2 className="text-lg font-bold mb-4 text-white">Event Moderation</h2>

        {/* Tabs */}
        <div className="flex bg-[#1E202A] rounded-full p-1 border border-gray-800 mb-6">
          {(['PENDING', 'APPROVED', 'REJECTED'] as TabStatus[]).map((tab) => {
            const isActive = activeTab === tab;
            return (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`flex-1 py-2.5 text-sm font-medium rounded-full transition-all flex justify-center items-center gap-2
                  ${isActive ? 'bg-[#E5B05C] text-black shadow-md' : 'text-gray-400 hover:text-white'}`}
              >
                {isActive && tab === 'PENDING' && <Check size={16} strokeWidth={3} />}
                {isActive && tab === 'APPROVED' && <Check size={16} strokeWidth={3} />}
                {tab.charAt(0) + tab.slice(1).toLowerCase()}
              </button>
            );
          })}
        </div>

        {/* Events List */}
        <div className="space-y-4">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-[#E5B05C]"></div>
            </div>
          ) : events.length === 0 ? (
            <p className="text-gray-500 text-center py-6 bg-[#1E202A] rounded-2xl border border-gray-800">
              No {activeTab.toLowerCase()} events found.
            </p>
          ) : (
            events.map((event) => (
              <div 
                key={event.id} 
                className="bg-[#1E202A] border border-[#2A2C38] rounded-2xl p-4 flex items-center justify-between"
              >
                <div className="flex items-center gap-4 flex-1">
                  {/* Calendar Icon Box */}
                  <div className="bg-[#2A2C38] p-3 rounded-xl flex-shrink-0">
                    <Calendar className="text-[#E5B05C]" size={24} />
                  </div>
                  
                  {/* Event Text Info */}
                  <div className="flex-1 min-w-0">
                    <h3 className="font-bold text-base text-white truncate">{event.title}</h3>
                    <p className="text-gray-400 text-xs mt-0.5 truncate">
                      {event.date || 'Date TBA'}
                    </p>
                    
                    {/* Status Pill */}
                    <div className="mt-1.5">
                      <span className={`text-[10px] px-2 py-0.5 rounded font-bold tracking-wide uppercase
                        ${event.status === 'APPROVED' ? 'bg-[#183321] text-[#4ade80]' : 
                          event.status === 'PENDING' ? 'bg-[#332415] text-[#fb923c]' : 
                          'bg-[#331515] text-[#f87171]'}`}
                      >
                        {event.status}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Actions: Approve/Reject buttons for PENDING, or Arrow for others */}
                <div className="ml-4 flex items-center flex-shrink-0">
                  {activeTab === 'PENDING' ? (
                    <div className="flex gap-2">
                      <button 
                        onClick={() => handleModerate(event.id, 'APPROVED')}
                        disabled={moderateMutation.isPending}
                        className="bg-green-600/20 text-green-500 hover:bg-green-600 hover:text-white p-2 rounded-lg transition-colors"
                        title="Approve"
                      >
                        <Check size={20} />
                      </button>
                      <button 
                        onClick={() => handleModerate(event.id, 'REJECTED')}
                        disabled={moderateMutation.isPending}
                        className="bg-red-600/20 text-red-500 hover:bg-red-600 hover:text-white p-2 rounded-lg transition-colors"
                        title="Reject"
                      >
                        <X size={20} />
                      </button>
                    </div>
                  ) : (
                    <button className="p-2 text-gray-500 hover:text-[#E5B05C] transition-colors">
                      {activeTab === 'APPROVED' ? <Check size={20} className="text-[#E5B05C]" /> : <ChevronRight size={20} />}
                    </button>
                  )}
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* ─── User Management Section (Matching your second screenshot) ─── */}
      <div className="mt-10 border-t border-gray-800 pt-8">
        <h2 className="text-lg font-bold mb-4 text-white">User Management</h2>
        <div className="relative">
          <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <Search size={18} className="text-[#E5B05C]" />
          </div>
          <input 
            type="text" 
            placeholder="Search by Email" 
            className="w-full bg-transparent border border-gray-600 text-white rounded-xl pl-12 pr-4 py-3.5 focus:outline-none focus:border-[#E5B05C] transition-colors placeholder-gray-500"
          />
        </div>
      </div>

    </div>
  );
}
