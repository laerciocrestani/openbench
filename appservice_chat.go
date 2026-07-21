package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/laerciocrestani/openbench/internal/desktop"
)

type pendingChatTool struct {
	id string
	ch chan bool
}

// ChatStream starts a streaming project chat in the background.
// model selects the model for this request (empty = config default).
// Emits: chat:chunk (string), chat:tool_request (ChatToolRequest),
// chat:done (ChatDonePayload), chat:error (string).
func (s *AppService) ChatStream(message string, history []desktop.ChatMessageView, model string) error {
	path := strings.TrimSpace(s.currentPath())
	if path == "" {
		return fmt.Errorf("abra um projeto para usar o chat")
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("mensagem vazia")
	}

	s.mu.Lock()
	if s.chatCancel != nil {
		s.chatCancel()
		s.chatCancel = nil
	}
	s.clearPendingToolLocked()
	ctx, cancel := context.WithCancel(context.Background())
	s.chatCancel = cancel
	appRef := s.app
	s.mu.Unlock()

	emit := func(event string, data any) {
		if appRef == nil {
			return
		}
		appRef.Event.Emit(event, data)
	}

	go func() {
		defer cancel()

		hooks := desktop.ChatAgentHooks{
			OnChunk: func(delta string) {
				if delta == "" {
					return
				}
				emit("chat:chunk", delta)
			},
			RequestTool: func(ctx context.Context, req desktop.ChatToolRequest) (bool, error) {
				return s.waitChatToolDecision(ctx, req, emit)
			},
			OnProjectMutated: func() {
				s.emitDashboardRefresh(path)
			},
		}

		done, err := desktop.RunProjectChatStream(ctx, path, history, message, model, hooks)

		s.mu.Lock()
		if s.chatCancel != nil {
			s.chatCancel = nil
		}
		s.clearPendingToolLocked()
		s.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				emit("chat:error", "chat cancelado")
				return
			}
			emit("chat:error", err.Error())
			return
		}
		emit("chat:done", done)
	}()

	return nil
}

// GetChatModels returns selectable models for the chat UI.
func (s *AppService) GetChatModels() (*desktop.ChatModelsView, error) {
	return desktop.LoadChatModels()
}

// GetAIConfig returns provider and Chat/Git model settings for the IA tab.
func (s *AppService) GetAIConfig() (*desktop.AIConfigView, error) {
	return desktop.LoadAIConfig()
}

// SaveAISettings persists provider, API key (optional) and Chat/Git models.
func (s *AppService) SaveAISettings(
	provider, apiKey, gitModel, gitFallback, chatModel, chatFallback string,
) error {
	return desktop.SaveAISettings(provider, apiKey, gitModel, gitFallback, chatModel, chatFallback)
}

// ChatCancel aborts the active chat stream, if any.
func (s *AppService) ChatCancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingTool != nil {
		select {
		case s.pendingTool.ch <- false:
		default:
		}
		s.pendingTool = nil
	}
	if s.chatCancel != nil {
		s.chatCancel()
		s.chatCancel = nil
	}
}

// ChatApproveTool approves a pending privileged tool request.
func (s *AppService) ChatApproveTool(id string) error {
	return s.resolveChatTool(id, true)
}

// ChatDenyTool denies a pending privileged tool request.
func (s *AppService) ChatDenyTool(id string) error {
	return s.resolveChatTool(id, false)
}

func (s *AppService) resolveChatTool(id string, approved bool) error {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingTool == nil {
		return fmt.Errorf("nenhuma tool pendente")
	}
	if id != "" && s.pendingTool.id != id {
		return fmt.Errorf("tool pendente não corresponde ao id")
	}
	select {
	case s.pendingTool.ch <- approved:
	default:
		return fmt.Errorf("decisão de tool já enviada")
	}
	s.pendingTool = nil
	return nil
}

func (s *AppService) waitChatToolDecision(
	ctx context.Context,
	req desktop.ChatToolRequest,
	emit func(event string, data any),
) (bool, error) {
	ch := make(chan bool, 1)

	s.mu.Lock()
	if s.pendingTool != nil {
		s.mu.Unlock()
		return false, fmt.Errorf("já existe uma tool aguardando aprovação")
	}
	s.pendingTool = &pendingChatTool{id: req.ID, ch: ch}
	s.mu.Unlock()

	emit("chat:tool_request", req)

	select {
	case approved := <-ch:
		return approved, nil
	case <-ctx.Done():
		s.mu.Lock()
		if s.pendingTool != nil && s.pendingTool.ch == ch {
			s.pendingTool = nil
		}
		s.mu.Unlock()
		return false, ctx.Err()
	}
}

func (s *AppService) clearPendingToolLocked() {
	if s.pendingTool == nil {
		return
	}
	select {
	case s.pendingTool.ch <- false:
	default:
	}
	s.pendingTool = nil
}
