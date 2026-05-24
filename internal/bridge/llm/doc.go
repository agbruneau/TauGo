// Package llm declares the narrow interface (Client) that TauGo consumes
// from any injected LLM client. No concrete LLM is embedded; the production
// implementation is wired in internal/app/. A deterministic stub is provided
// for CI and calibration reproducibility (cf. PRD §12.2).
package llm
