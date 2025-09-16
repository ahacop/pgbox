# ADR-001: Nix Flake Versioning Strategy

## Status
Accepted

## Date
2025-01-16

## Context

When running `nix run github:ahacop/pgbox -- --version`, the application was returning "dev" instead of the proper version number. This occurred because the Nix flake build wasn't setting the version ldflags that the Go binary expects.

The application uses:
- Build-time ldflags to embed version information (main.go:14-16)
- Charmbracelet/fang for CLI framework which handles --version flag
- Multiple build methods: Makefile (for development) and Nix flake (for distribution)

### Research Findings

We researched how other Go projects handle versioning in Nix flakes and found common patterns:

1. **Static versions**: Hardcoded in flake.nix
2. **Git-based versioning**: Using `self.shortRev` for commit info
3. **VERSION file**: External file read by both build systems
4. **Modern approach**: Using `ldflags` attribute instead of custom buildPhase

Popular projects like Tailscale, Headscale, and nix-search-cli use git-based versioning with automatic commit injection.

## Decision

Implement git-based versioning in flake.nix using:
- `buildGoModule rec` to allow self-referencing attributes
- Dynamic version: `if (self ? shortRev) then "0.1.0-${self.shortRev}" else "0.1.0"`
- Standard `ldflags` attribute instead of custom buildPhase
- Pass version, commit, and date via ldflags

## Implementation

```nix
packages.default = pkgs.buildGoModule rec {
  pname = "pgbox";
  version = if (self ? shortRev) then "0.1.0-${self.shortRev}" else "0.1.0";

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
    "-X main.commit=${self.rev or "unknown"}"
    "-X main.date=1970-01-01T00:00:00Z"  # Reproducible builds
  ];
  # ...
};
```

## Consequences

### Positive
- Version correctly displays when running via `nix run`
- Git commits automatically included for traceability
- Follows Go/Nix ecosystem best practices
- Uses buildGoModule's built-in support (no custom phases)
- Single place to update version in flake.nix

### Negative
- Version in flake.nix separate from git tags
- Manual update required when cutting releases
- Two sources of truth (git tags for Makefile, flake.nix for Nix)

### Maintenance Burden
- **Minimal**: Update version on line 20 of flake.nix when releasing
- Git metadata handled automatically
- No custom build logic to maintain

## Alternatives Considered

1. **Hardcoded static version**: Simple but no git traceability
2. **VERSION file**: Would unify both build systems but adds another file
3. **Generate flake.nix**: Too complex for current needs
4. **Custom buildPhase**: Works but not idiomatic for modern Nix

## Future Improvements

If the dual maintenance becomes problematic, consider:
1. Creating a VERSION file read by both build systems
2. Automation script to update flake.nix version from git tags
3. Moving to a single build system

## References
- [Nix buildGoModule documentation](https://nixos.org/manual/nixpkgs/stable/#ssec-go-modules)
- [Headscale flake.nix](https://github.com/juanfont/headscale)
- [Tailscale flake.nix](https://github.com/tailscale/tailscale)