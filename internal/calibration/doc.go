// Package calibration manages adaptive thresholds and weighted profiles
// for the τ operator. Profiles are versioned, persisted to disk, and
// invalidated on hardware/model/corpus drift (cf. PRD §11).
package calibration
