package channel

import (
	"context"
	"log/slog"
	"os"
	"testing"

	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
)

func getListLength(list *listNode) int {
	if list == nil {
		return 0
	}
	next := list.next
	count := 0
	for next != nil {
		count++
		next = next.next
	}
	return count
}

func TestSaveContext(t *testing.T) {
	r, w, _ := os.Pipe()
	channel := NewChannel(w, r, slog.Default())
	defer channel.Close(context.Background())

	ctx := context.WithValue(context.Background(), "key", "value")

	cleanup1 := channel.maySaveContextLocked(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_PAUSE,
		HandlerId: "handlerId1",
	})
	cleanup2 := channel.maySaveContextLocked(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_PAUSE,
		HandlerId: "handlerId2",
	})

	origCtx := UnwrapContext(channel.getContext(methodEventMap[FbsRequest.MethodPRODUCER_PAUSE]), "handlerId1")
	if ctx.Value("key") != origCtx.Value("key") {
		t.Errorf("Expected context to be the same")
	}

	cleanup1()

	if length := getListLength(channel.contextList); length != 1 {
		t.Errorf("Expected list length to be 1, got %d", length)
	}

	origCtx = UnwrapContext(channel.getContext(methodEventMap[FbsRequest.MethodPRODUCER_PAUSE]), "handlerId2")
	if ctx.Value("key") != origCtx.Value("key") {
		t.Errorf("Expected context to be the same")
	}

	cleanup2()

	if length := getListLength(channel.contextList); length != 0 {
		t.Errorf("Expected list length to be 0, got %d", length)
	}
}
