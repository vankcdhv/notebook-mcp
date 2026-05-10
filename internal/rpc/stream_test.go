package rpc

import "testing"

func TestDecodeChatChunksExtractsIncrementalText(t *testing.T) {
	response := `)]}'
123
[["wrb.fr",null,"[[\"hello\",null,[\"conv-id\"],null,[null,null,null,null,1]]]",null,null,null,"generic"]]
456
[["wrb.fr",null,"[[\"hello world\",null,[\"conv-id\"],null,[null,null,null,null,1]]]",null,null,null,"generic"]]`

	chunks, convID, refs, err := DecodeChatChunks(response)
	if err != nil {
		t.Fatal(err)
	}
	if convID != "conv-id" {
		t.Fatalf("convID = %q", convID)
	}
	if len(refs) != 0 {
		t.Fatalf("refs len = %d, want 0", len(refs))
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks len = %d, want 2", len(chunks))
	}
	if chunks[0].Text != "hello" || chunks[0].Seq != 1 || chunks[0].IsFinal {
		t.Fatalf("first chunk = %+v", chunks[0])
	}
	if chunks[1].Text != "hello world" || chunks[1].Seq != 2 || !chunks[1].IsFinal {
		t.Fatalf("second chunk = %+v", chunks[1])
	}
}
