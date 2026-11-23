-- +goose Up
-- NVIDIA's hosted API catalog (build.nvidia.com) joins the AI providers --
-- see infrastructure/ai/nvidia.go. Backward compatible: older binaries never
-- write 'nvidia', and every value they do write stays valid.
ALTER TABLE ai_connections DROP CONSTRAINT ai_connections_provider_check;
ALTER TABLE ai_connections ADD CONSTRAINT ai_connections_provider_check
    CHECK (provider IN ('openai', 'anthropic', 'gemini', 'ollama', 'nvidia'));

-- +goose Down
-- Fails if a tenant is still on NVIDIA -- switch them to another provider
-- (or delete the row) before rolling back.
ALTER TABLE ai_connections DROP CONSTRAINT ai_connections_provider_check;
ALTER TABLE ai_connections ADD CONSTRAINT ai_connections_provider_check
    CHECK (provider IN ('openai', 'anthropic', 'gemini', 'ollama'));
