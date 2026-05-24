import { useMemo, useState } from 'react';
import { ChatMessageData } from '../contracts';
import { Button } from './ui/button';

interface TripChatPanelProps {
  title: string;
  tripID: string;
  currentUserID: string;
  peerLabel?: string;
  messages: ChatMessageData[];
  onSend: (text: string) => void;
}

export const TripChatPanel = ({
  title,
  tripID,
  currentUserID,
  peerLabel,
  messages,
  onSend,
}: TripChatPanelProps) => {
  const [text, setText] = useState('');

  const scopedMessages = useMemo(
    () => messages.filter((m) => m.tripID === tripID),
    [messages, tripID],
  );

  const handleSend = () => {
    const trimmed = text.trim();
    if (!trimmed) return;
    onSend(trimmed);
    setText('');
  };

  return (
    <div className="flex flex-col gap-3 border rounded-md p-3">
      <h4 className="text-sm font-semibold">{title}</h4>

      <div className="max-h-40 overflow-y-auto flex flex-col gap-2">
        {scopedMessages.length === 0 && (
          <p className="text-xs text-gray-500">No messages yet.</p>
        )}
        {scopedMessages.map((msg) => (
          <div key={msg.messageID ?? `${msg.senderID}-${msg.sentAt}-${msg.text}`} className="text-xs">
            <span className="font-medium">
              {msg.senderID === currentUserID ? 'You' : peerLabel ?? msg.senderID}
            </span>
            {': '}
            <span>{msg.text}</span>
          </div>
        ))}
      </div>

      <div className="flex gap-2">
        <input
          className="flex-1 border rounded px-2 py-1 text-sm"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSend()}
          placeholder="Type a message"
        />
        <Button onClick={handleSend}>Send</Button>
      </div>
    </div>
  );
};
