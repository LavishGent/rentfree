# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multi-layer caching with memory (bigcache) and Redis backends
- Resilience patterns: circuit breaker, retry with exponential backoff, bulkhead
- Metrics tracking with pluggable publishers (logging, custom)
- Graceful degradation when Redis fails
- Cache-aside pattern via `GetOrCreate` method
- Comprehensive configuration via JSON files with environment variable overrides
- Health check API for monitoring cache status
- Batch operations: `GetMany`, `SetMany`, `DeleteMany`
- Pattern-based cache clearing with wildcard support
- Flexible cache levels: MemoryOnly, RedisOnly, MemoryThenRedis, All
- Priority-based eviction for memory cache entries
- Fire-and-forget async Redis writes
- Minimal dependencies (only bigcache and go-redis)

### Known Limitations
- Memory cache uses global TTL (LifeWindow) - per-entry TTL only works with Redis layer
- BigCache doesn't support true per-entry TTL; all entries share the configured LifeWindow

### Documentation
- Comprehensive README with architecture diagrams
- Working example in `examples/basic/`
- Test coverage: Config (97.5%), Metrics (92.7%), Resilience (90.1%)

## [0.1.0] - 2025-12-29

### Initial Release
- First developer preview release for internal testing
- Core caching functionality stable
- Ready for developer feedback and testing

[Unreleased]: https://github.com/darrell-green/rentfree/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/darrell-green/rentfree/releases/tag/v0.1.0
