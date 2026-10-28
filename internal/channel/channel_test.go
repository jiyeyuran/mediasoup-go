package channel

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	flatbuffers "github.com/google/flatbuffers/go"
	FbsMessage "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Message"
	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
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

func TestPendingRequestsAndObserver(t *testing.T) {
	var (
		mu    sync.Mutex
		stats []RequestStats
	)
	observed := func() []RequestStats {
		mu.Lock()
		defer mu.Unlock()

		return append([]RequestStats(nil), stats...)
	}

	r, w, _ := os.Pipe()
	// Nothing drains the read end, so every request stays pending until its
	// context is cancelled.
	channel := NewChannel(w, r, slog.Default(), slog.Default(), WithRequestObserver(func(s RequestStats) {
		mu.Lock()
		defer mu.Unlock()

		stats = append(stats, s)
	}))
	defer channel.Close(context.Background())

	if got := channel.PendingRequests(); got != 0 {
		t.Fatalf("PendingRequests() = %d, want 0", got)
	}

	ctx, cancel := context.WithCancel(context.Background())

	const requests = 4
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			//nolint:errcheck // The request is expected to be abandoned.
			channel.Request(ctx, &FbsRequest.RequestT{Method: FbsRequest.MethodWORKER_DUMP})
		}()
	}

	deadline := time.After(5 * time.Second)
	for channel.PendingRequests() != requests {
		select {
		case <-deadline:
			t.Fatalf("PendingRequests() = %d, want %d", channel.PendingRequests(), requests)
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	wg.Wait()

	if got := channel.PendingRequests(); got != 0 {
		t.Fatalf("PendingRequests() after cancel = %d, want 0", got)
	}

	reported := observed()
	if len(reported) != requests {
		t.Fatalf("observer saw %d requests, want %d", len(reported), requests)
	}

	var sawPending bool
	for _, s := range reported {
		if s.Err == nil {
			t.Errorf("request reported no error, want the cancellation")
		}
		if s.Method != FbsRequest.MethodWORKER_DUMP {
			t.Errorf("Method = %v, want WORKER_DUMP", s.Method)
		}
		if s.HandlerID != DefaultHandlerID {
			t.Errorf("HandlerID = %q, want %q", s.HandlerID, DefaultHandlerID)
		}
		if s.Pending > 0 {
			sawPending = true
		}
	}
	// The requests were all in flight together, so whichever was reported first
	// must have seen the others still waiting.
	if !sawPending {
		t.Error("no request saw the concurrent ones in its pending count")
	}
}

// Close used to walk responsesCh unlocked while requests giving up deleted their
// own entries, which is a concurrent map read and write: a panic, not just a race.
// This is what a worker dying with requests in flight looks like.
func TestCloseWithRequestsInFlight(t *testing.T) {
	r, w, _ := os.Pipe()
	// Nothing answers, so the requests are still in flight when Close runs.
	channel := NewChannel(w, r, slog.Default(), slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const requests = 64
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			//nolint:errcheck // The request is expected to be abandoned.
			channel.Request(ctx, &FbsRequest.RequestT{Method: FbsRequest.MethodWORKER_DUMP})
		}()
	}

	deadline := time.After(5 * time.Second)
	for channel.PendingRequests() < requests {
		select {
		case <-deadline:
			t.Fatalf("PendingRequests() = %d, want %d", channel.PendingRequests(), requests)
		case <-time.After(time.Millisecond):
		}
	}

	// Racing the two against each other: Close walks the map while cancel makes
	// every request delete its entry.
	go cancel()
	channel.Close(context.Background())

	wg.Wait()

	if got := channel.PendingRequests(); got != 0 {
		t.Errorf("PendingRequests() after close = %d, want 0", got)
	}
}

func TestSaveContext(t *testing.T) {
	r, w, _ := os.Pipe()
	channel := NewChannel(w, r, slog.Default(), slog.Default())
	defer channel.Close(context.Background())

	ctx := context.TODO()

	cleanup1 := channel.maySaveContextLocked(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_PAUSE,
		HandlerId: "handlerId1",
	})
	cleanup2 := channel.maySaveContextLocked(ctx, &FbsRequest.RequestT{
		Method:    FbsRequest.MethodPRODUCER_PAUSE,
		HandlerId: "handlerId2",
	})

	origCtx := UnwrapContext(channel.getContext(needContextMethodEvents[FbsRequest.MethodPRODUCER_PAUSE]), "handlerId1")
	if ctx.Value("key") != origCtx.Value("key") {
		t.Errorf("Expected context to be the same")
	}

	cleanup1()

	if length := getListLength(channel.contextList); length != 1 {
		t.Errorf("Expected list length to be 1, got %d", length)
	}

	origCtx = UnwrapContext(channel.getContext(needContextMethodEvents[FbsRequest.MethodPRODUCER_PAUSE]), "handlerId2")
	if ctx.Value("key") != origCtx.Value("key") {
		t.Errorf("Expected context to be the same")
	}

	cleanup2()

	if length := getListLength(channel.contextList); length != 0 {
		t.Errorf("Expected list length to be 0, got %d", length)
	}
}

func TestPackNotificationAlwaysWritesHandlerId(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)
	payload := packNotification(builder, &FbsNotification.NotificationT{Event: FbsNotification.EventWORKER_CLOSE})

	msg := FbsMessage.GetRootAsMessage(payload, 0)
	require.Equal(t, FbsMessage.BodyNotification, msg.DataType())

	var table flatbuffers.Table
	require.True(t, msg.Data(&table))
	var notification FbsNotification.Notification
	notification.Init(table.Bytes, table.Pos)

	require.NotNil(t, notification.HandlerId(), "handler_id is required by the schema")
	require.Equal(t, "", string(notification.HandlerId()))
	require.Equal(t, FbsNotification.EventWORKER_CLOSE, notification.Event())
}