package messagehandler

import (
	"context"
	"fmt"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// sendReplyMessage sends a text message reply, optionally quoting another message
func (h *WhatsMeowEventHandler) sendReplyMessage(chatJID types.JID, replyText string, quotedMsgID string) error {
	// Regular message (no quoting)
	if quotedMsgID == "" {
		msg := &waProto.Message{
			Conversation: proto.String(replyText),
		}
		_, err := h.client.SendMessage(context.Background(), chatJID, msg)
		if err != nil {
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	}

	// Quoted message reply
	senderJID := chatJID
	if chatJID.Server == "g.us" {
		// In groups, we need to get the actual participant JID
		// Since we don't have direct access to the original sender's JID here,
		// we'll use a placeholder that works with WhatsApp
		senderJID = types.NewJID(chatJID.User, "s.whatsapp.net")
	}

	// Create the message with proper quoting
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(replyText),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(quotedMsgID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	_, err := h.client.SendMessage(context.Background(), chatJID, msg)
	if err != nil {
		return fmt.Errorf("failed to send quoted reply: %w", err)
	}
	return nil
}

// editMessageContent edits any type of message content (text or media caption)
func (h *WhatsMeowEventHandler) editMessageContent(chatJID types.JID, messageID string, newContent string, originalMsg *waProto.Message) error {
	var msg *waProto.Message
	var mediaType string

	// Handle different message types differently
	if originalMsg == nil {
		// Simple text message edit (no original message provided)
		msg = &waProto.Message{
			Conversation: proto.String(newContent),
		}
		mediaType = "text"
	} else if originalMsg.GetImageMessage() != nil {
		// Image message with caption
		msg = &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				URL:           proto.String(originalMsg.GetImageMessage().GetURL()),
				Mimetype:      proto.String(originalMsg.GetImageMessage().GetMimetype()),
				Caption:       proto.String(newContent),
				FileEncSHA256: originalMsg.GetImageMessage().GetFileEncSHA256(),
				FileSHA256:    originalMsg.GetImageMessage().GetFileSHA256(),
				FileLength:    proto.Uint64(originalMsg.GetImageMessage().GetFileLength()),
				Height:        proto.Uint32(originalMsg.GetImageMessage().GetHeight()),
				Width:         proto.Uint32(originalMsg.GetImageMessage().GetWidth()),
				MediaKey:      originalMsg.GetImageMessage().GetMediaKey(),
				DirectPath:    proto.String(originalMsg.GetImageMessage().GetDirectPath()),
			},
		}
		mediaType = "image"
	} else if originalMsg.GetVideoMessage() != nil {
		// Video message with caption
		msg = &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				URL:           proto.String(originalMsg.GetVideoMessage().GetURL()),
				Mimetype:      proto.String(originalMsg.GetVideoMessage().GetMimetype()),
				Caption:       proto.String(newContent),
				FileEncSHA256: originalMsg.GetVideoMessage().GetFileEncSHA256(),
				FileSHA256:    originalMsg.GetVideoMessage().GetFileSHA256(),
				FileLength:    proto.Uint64(originalMsg.GetVideoMessage().GetFileLength()),
				Seconds:       proto.Uint32(originalMsg.GetVideoMessage().GetSeconds()),
				MediaKey:      originalMsg.GetVideoMessage().GetMediaKey(),
				DirectPath:    proto.String(originalMsg.GetVideoMessage().GetDirectPath()),
			},
		}
		mediaType = "video"
	} else if originalMsg.GetDocumentMessage() != nil {
		// Document message with caption
		msg = &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				URL:           proto.String(originalMsg.GetDocumentMessage().GetURL()),
				Mimetype:      proto.String(originalMsg.GetDocumentMessage().GetMimetype()),
				Title:         proto.String(originalMsg.GetDocumentMessage().GetTitle()),
				FileName:      proto.String(originalMsg.GetDocumentMessage().GetFileName()),
				FileEncSHA256: originalMsg.GetDocumentMessage().GetFileEncSHA256(),
				FileSHA256:    originalMsg.GetDocumentMessage().GetFileSHA256(),
				FileLength:    proto.Uint64(originalMsg.GetDocumentMessage().GetFileLength()),
				MediaKey:      originalMsg.GetDocumentMessage().GetMediaKey(),
				DirectPath:    proto.String(originalMsg.GetDocumentMessage().GetDirectPath()),
				Caption:       proto.String(newContent),
			},
		}
		mediaType = "document"
	} else {
		// Extended text or other message types - convert to simple text
		msg = &waProto.Message{
			Conversation: proto.String(newContent),
		}
		mediaType = "text"
	}

	// Build and send the edit
	editMsg := h.client.BuildEdit(chatJID, types.MessageID(messageID), msg)
	resp, err := h.client.SendMessage(context.Background(), chatJID, editMsg)
	if err != nil {
		return fmt.Errorf("failed to send %s edit: %w", mediaType, err)
	}

	fmt.Printf("%s edited successfully. Server timestamp: %s\n", mediaType, resp.Timestamp)
	return nil
}

func (h *WhatsMeowEventHandler) SendResponse(msgInfo types.MessageInfo, response string) {
	fmt.Printf("Sending response to %s (isFromMe: %v, ID: %s)\n",
		msgInfo.Chat.String(), msgInfo.IsFromMe, msgInfo.ID)

	if msgInfo.IsFromMe {
		if err := h.editMessageContent(msgInfo.Chat, msgInfo.ID, response, nil); err != nil {
			fmt.Println("Edit failed:", err)
		}
	} else {
		// Send regular message (no quoting) to keep original YouTube preview visible
		if err := h.sendReplyMessage(msgInfo.Chat, response, ""); err != nil {
			fmt.Println("Reply failed:", err)
		}
	}
}
