package rpc

import "testing"

func TestDecodeResponse(t *testing.T) {
	response := `)]}'
[["wrb.fr","wXbhsf","[[[\"Title\",null,\"nb-id\"]]]",null,null,null,"generic"]]`
	got, err := DecodeResponse(response, "wXbhsf", false)
	if err != nil {
		t.Fatal(err)
	}
	outer := got.([]any)
	first := outer[0].([]any)
	nb := first[0].([]any)
	if nb[2] != "nb-id" {
		t.Fatalf("id = %v", nb[2])
	}
}

func TestDecodeChatResponse(t *testing.T) {
	response := `)]}'
123
[["wrb.fr",null,"[[\"answer text\",null,[\"conv-id\"],null,[null,null,null,null,1]]]",null,null,null,"generic"]]`
	answer, convID, _, err := DecodeChatResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "answer text" {
		t.Fatalf("answer = %q", answer)
	}
	if convID != "conv-id" {
		t.Fatalf("convID = %q", convID)
	}
}
