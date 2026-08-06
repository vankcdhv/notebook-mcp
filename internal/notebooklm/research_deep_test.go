package notebooklm

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/vankcdhv/notebook-mcp/internal/rpc"
)

const probeReportMarkdown = "# Green Tea GC\n\nThe collector reorganises spans.\n"

// protoField encodes one length-delimited protobuf field, matching the nesting
// NotebookLM uses for the deep research report blob.
func protoField(number int, payload []byte) []byte {
	out := binary.AppendUvarint(nil, uint64(number)<<3|2)
	out = binary.AppendUvarint(out, uint64(len(payload)))
	return append(out, payload...)
}

func deepReportBlob(markdown string) string {
	body := protoField(3, protoField(1, []byte(markdown)))
	return base64.StdEncoding.EncodeToString(protoField(1, body))
}

// deepResearchRPC replays a finished deep research task: its own report shows up
// among the found entries without a URL, and the report body arrives separately
// as a base64 protobuf blob in the trailing element.
type deepResearchRPC struct {
	statusCode float64
	blob       any
}

func (f *deepResearchRPC) Call(ctx context.Context, method string, params []any, sourcePath string, allowNull bool) (any, error) {
	found := []any{
		[]any{nil, "Green Tea GC report", nil, float64(5), nil, nil, nil},
		[]any{"https://go.dev/blog/greenteagc", "Green Tea GC", "snippet", float64(1)},
		[]any{"https://go.dev/doc/gc-guide", "", "snippet", float64(1)},
	}
	task := []any{
		"task-deep",
		[]any{
			"nb",
			[]any{"query", float64(1)},
			float64(5),
			[]any{found},
			f.statusCode,
			[]any{"report-id", f.blob, float64(5), nil, "deep_research.flash.prod"},
		},
		[]any{float64(1785344748), float64(246650000)},
	}
	return []any{[]any{task}}, nil
}

func (f *deepResearchRPC) Ask(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) (string, string, []rpc.ChatReference, error) {
	return "", "", nil, nil
}

func (f *deepResearchRPC) AskStream(ctx context.Context, notebookID, question string, sourceIDs []string, conversationID string) ([]rpc.ChatChunk, string, []rpc.ChatReference, error) {
	return nil, "", nil, nil
}

func (f *deepResearchRPC) UploadFile(ctx context.Context, notebookID, sourceID, filePath, mimeType string) error {
	return nil
}

func TestPollResearchSeparatesReportFromWebResults(t *testing.T) {
	client := New(&deepResearchRPC{statusCode: 2, blob: deepReportBlob(probeReportMarkdown)})

	results, err := client.PollResearch(context.Background(), "nb")
	if err != nil {
		t.Fatal(err)
	}

	var web, reports []ResearchResult
	for _, r := range results {
		if r.URL == "" && r.Type != "report" {
			t.Fatalf("web result without a url would poison the import batch: %+v", r)
		}
		if r.Type == "report" {
			reports = append(reports, r)
			continue
		}
		web = append(web, r)
	}

	// The untitled page stays importable; only the URL-less report entry is dropped.
	if len(web) != 2 {
		t.Fatalf("web results = %d, want 2: %+v", len(web), web)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	report := reports[0]
	if report.Content != probeReportMarkdown {
		t.Fatalf("report content = %q", report.Content)
	}
	if report.Title != "Green Tea GC" {
		t.Fatalf("report title = %q, want the leading heading", report.Title)
	}
	if report.TaskID != "task-deep" || report.Status != "completed" {
		t.Fatalf("report = %+v", report)
	}
}

func TestPollResearchReportsFinishedDeepTaskAsCompleted(t *testing.T) {
	for _, tc := range []struct {
		name string
		code float64
		want string
	}{
		{"running", 1, "in_progress"},
		{"finished", 2, "completed"},
		{"finished and imported", 6, "completed"},
		{"finished deep variant", 5, "completed"},
		{"finished deep variant", 7, "completed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := New(&deepResearchRPC{statusCode: tc.code, blob: deepReportBlob(probeReportMarkdown)})
			results, err := client.PollResearch(context.Background(), "nb")
			if err != nil {
				t.Fatal(err)
			}
			if len(results) == 0 {
				t.Fatal("no results")
			}
			if results[0].Status != tc.want {
				t.Fatalf("status = %q, want %q", results[0].Status, tc.want)
			}
		})
	}
}

func TestPollResearchOmitsReportWhileItIsGenerated(t *testing.T) {
	client := New(&deepResearchRPC{statusCode: 1, blob: nil})

	results, err := client.PollResearch(context.Background(), "nb")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Type == "report" {
			t.Fatalf("unfinished report was returned: %+v", r)
		}
	}
}

func TestImportResearchRejectsResultsItCannotEncode(t *testing.T) {
	rpcClient := &recordingRPC{result: []any{}}
	client := New(rpcClient)

	_, err := client.ImportResearch(context.Background(), "nb-id", "task-id", []ResearchResult{
		{Title: "Report without a body", Type: "report"},
		{Title: "Page without a url"},
	})
	if err == nil {
		t.Fatal("ImportResearch returned nil error for unusable results")
	}
	if rpcClient.method != "" {
		t.Fatalf("a batch NotebookLM would reject was still sent: %q", rpcClient.method)
	}
}

func TestStartDeepResearchReadsTheTaskID(t *testing.T) {
	// Deep research answers with the report ID first and the task ID second.
	rpcClient := &recordingRPC{result: []any{"report-id", "task-uuid"}}
	client := New(rpcClient)

	result, err := client.StartResearch(context.Background(), "nb-id", "query", "web", "deep")
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "task-uuid" {
		t.Fatalf("task id = %q, want the task uuid rather than the report id", result.TaskID)
	}
	if result.Status != "in_progress" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestStartFastResearchReadsTheTaskID(t *testing.T) {
	rpcClient := &recordingRPC{result: []any{"task-uuid"}}
	client := New(rpcClient)

	result, err := client.StartResearch(context.Background(), "nb-id", "query", "web", "fast")
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "task-uuid" {
		t.Fatalf("task id = %q", result.TaskID)
	}
}

func TestDecodeResearchReportIgnoresUnrelatedBlobs(t *testing.T) {
	if got := decodeResearchReport(""); got != "" {
		t.Fatalf("empty blob decoded to %q", got)
	}
	if got := decodeResearchReport("not base64 !!"); got != "" {
		t.Fatalf("invalid blob decoded to %q", got)
	}
	plan := base64.StdEncoding.EncodeToString(protoField(1, protoField(1, []byte("(1) research plan"))))
	if got := decodeResearchReport(plan); got != "" {
		t.Fatalf("plan entry decoded as a report: %q", got)
	}
}

func TestReportTitleFallsBackToTheFirstLine(t *testing.T) {
	if got := reportTitle("\n\nPlain first line\nsecond\n"); got != "Plain first line" {
		t.Fatalf("title = %q", got)
	}
	if got := reportTitle(strings.Repeat("\n", 3)); got != "" {
		t.Fatalf("title = %q, want empty", got)
	}
}
