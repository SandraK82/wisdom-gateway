package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/thoughtracker/shared-wisdom/internal/models"
)

// VerifyEd25519 verifies an Ed25519 signature.
// publicKeyBase64: base64-encoded 32-byte public key
// data: the signed data
// signatureBase64: base64-encoded 64-byte signature
func VerifyEd25519(publicKeyBase64 string, data []byte, signatureBase64 string) (bool, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return false, fmt.Errorf("invalid public key base64: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size: %d (expected %d)", len(pubKeyBytes), ed25519.PublicKeySize)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, fmt.Errorf("invalid signature base64: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return false, nil
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)
	return ed25519.Verify(pubKey, data, sigBytes), nil
}

// addressToCanonical converts a models.Address to a canonical map representation.
func addressToCanonical(addr models.Address) interface{} {
	if addr.IsEmpty() {
		return nil
	}
	return map[string]interface{}{
		"domain":      string(addr.Domain),
		"entity":      addr.Entity,
		"server_port": addr.ServerPort,
	}
}

// addressSliceToCanonical converts a slice of addresses to canonical representation.
func addressSliceToCanonical(addrs []models.Address) []interface{} {
	result := make([]interface{}, len(addrs))
	for i, a := range addrs {
		result[i] = addressToCanonical(a)
	}
	return result
}

// VerifyFragment verifies a fragment's Ed25519 signature using canonical JSON.
func VerifyFragment(fragment *models.Fragment, publicKey string) error {
	transformVal := addressToCanonical(fragment.Transform)

	tags := addressSliceToCanonical(fragment.Tags)

	fields := map[string]interface{}{
		"confidence":    fragment.Confidence,
		"content":       fragment.Content,
		"creator":       addressToCanonical(fragment.Creator),
		"evidence_type": string(fragment.EvidenceType),
		"tags":          tags,
		"transform":     transformVal,
		"uuid":          fragment.UUID,
		"when":          fragment.When.Format("2006-01-02T15:04:05.000Z"),
	}

	return verifyEntity(fields, publicKey, fragment.Signature, "fragment")
}

// VerifyRelation verifies a relation's Ed25519 signature using canonical JSON.
func VerifyRelation(relation *models.Relation, publicKey string) error {
	fields := map[string]interface{}{
		"by":      addressToCanonical(relation.By),
		"content": relation.Content,
		"creator": addressToCanonical(relation.Creator),
		"from":    addressToCanonical(relation.From),
		"to":      addressToCanonical(relation.To),
		"type":    string(relation.Type),
		"uuid":    relation.UUID,
		"when":    relation.When.Format("2006-01-02T15:04:05.000Z"),
	}

	return verifyEntity(fields, publicKey, relation.Signature, "relation")
}

// VerifyTag verifies a tag's Ed25519 signature using canonical JSON.
func VerifyTag(tag *models.Tag, publicKey string) error {
	fields := map[string]interface{}{
		"category": string(tag.Category),
		"content":  tag.Content,
		"creator":  addressToCanonical(tag.Creator),
		"name":     tag.Name,
		"uuid":     tag.UUID,
	}

	return verifyEntity(fields, publicKey, tag.Signature, "tag")
}

// VerifyTransform verifies a transform's Ed25519 signature using canonical JSON.
func VerifyTransform(transform *models.Transform, publicKey string) error {
	tags := addressSliceToCanonical(transform.Tags)

	fields := map[string]interface{}{
		"additional_data": transform.AdditionalData,
		"agent":           addressToCanonical(transform.Agent),
		"description":     transform.Description,
		"name":            transform.Name,
		"tags":            tags,
		"transform_from":  transform.TransformFrom,
		"transform_to":    transform.TransformTo,
		"uuid":            transform.UUID,
	}

	return verifyEntity(fields, publicKey, transform.Signature, "transform")
}

// VerifyAgent verifies an agent's Ed25519 signature using canonical JSON.
func VerifyAgent(agent *models.Agent, publicKey string) error {
	// Trust is part of the signature — new trust = new version + new signature
	trustMap := map[string]interface{}{
		"num_trusts": agent.Trust.NumTrusts,
		"trusts":     agent.Trust.Trusts,
	}
	// Ensure trusts is an empty array, not null
	if agent.Trust.Trusts == nil {
		trustMap["trusts"] = []interface{}{}
	}

	fields := map[string]interface{}{
		"description": agent.Description,
		"primary_hub": agent.PrimaryHub,
		"public_key":  agent.PublicKey,
		"trust":       trustMap,
		"uuid":        agent.UUID,
	}

	return verifyEntity(fields, publicKey, agent.Signature, "agent")
}

// verifyEntity is the common verification logic for all entity types.
func verifyEntity(fields map[string]interface{}, publicKey, signature, entityType string) error {
	canonical, err := CanonicalJSON(fields)
	if err != nil {
		return fmt.Errorf("failed to create canonical JSON for %s: %w", entityType, err)
	}

	valid, err := VerifyEd25519(publicKey, []byte(canonical), signature)
	if err != nil {
		return fmt.Errorf("signature verification error for %s: %w", entityType, err)
	}

	if !valid {
		return fmt.Errorf("invalid %s signature", entityType)
	}

	return nil
}
