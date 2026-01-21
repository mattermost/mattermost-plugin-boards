# Changelog: Bundle Size Optimization

## 2026-01-21: Optimization for Self-Hosted Single-Platform Deployment

### Summary

Optimized plugin bundle from multi-platform (~150-160 MB) to single-platform Linux AMD64 (~46 MB) for self-hosted deployment.

### Changes

#### Build System
- **Changed:** `.github/workflows/release.yml` now uses `make dist-linux` instead of `make dist`
- **Result:** Bundle contains only Linux AMD64 binary instead of all platforms
- **Size reduction:** ~150-160 MB → ~46 MB (~3.5x smaller)

#### Documentation Updates
- **RELEASE.md:** Updated platform information and bundle size
- **QUICKSTART-RELEASE.md:** Updated bundle size reference
- **PROJECT-SUMMARY.md:** Updated build output description
- **docs/AUTO-UPDATE-GUIDE.md:** Clarified single-platform approach

### Rationale

#### Why Single Platform is Sufficient

1. **Mattermost Plugin Loading Behavior:**
   - Mattermost loads only ONE binary from the plugin archive
   - Binary is selected based on server's OS/ARCH
   - Other platform binaries are completely ignored

2. **Self-Hosted Deployment:**
   - Single server = single architecture
   - No need for multiple platform binaries
   - Standard practice for self-hosted installations

3. **Official Compatibility:**
   - Officially acceptable approach
   - Fully compatible with Mattermost API upload
   - Works with auto-update through UI

### Benefits

- ✅ **Smaller bundle:** ~46 MB vs ~150-160 MB
- ✅ **Faster builds:** Only one platform to compile
- ✅ **Faster uploads:** Less data to transfer to server
- ✅ **Faster deployments:** Quicker download and extraction
- ✅ **Same functionality:** Auto-update still works perfectly

### Technical Details

#### What's Included in Bundle

```
boards-{version}.tar.gz (~46 MB)
├── plugin.json
├── server/
│   └── dist/
│       └── plugin-linux-amd64
└── webapp/
    └── dist/
        └── main.js
```

#### What's Excluded

- `plugin-linux-arm64`
- `plugin-darwin-amd64`
- `plugin-darwin-arm64`
- `plugin-windows-amd64.exe`

### Compatibility

- ✅ **Mattermost API upload:** Works perfectly
- ✅ **Auto-update via UI:** Works perfectly
- ✅ **Manual installation:** Works perfectly
- ✅ **Hot reload:** Works perfectly

### Migration Notes

#### For Users

No action required. The plugin will continue to work exactly as before.

#### For Developers

If you need to build for other platforms:

```bash
# Build all platforms
make dist

# Build specific platform
make dist-linux    # Linux AMD64 only
make dist-darwin   # macOS only (if target exists)
```

### Previous Approach (Multi-Platform)

**Pros:**
- Universal bundle works on any platform
- Convenient for distribution to multiple servers

**Cons:**
- Large bundle size (~150-160 MB)
- Slower builds (5 platforms to compile)
- Slower uploads and deployments
- Unnecessary for single-server deployments

### Current Approach (Single-Platform)

**Pros:**
- Optimized bundle size (~46 MB)
- Faster builds (1 platform to compile)
- Faster uploads and deployments
- Perfect for self-hosted single-server

**Cons:**
- Not universal (Linux AMD64 only)
- Need to rebuild for other platforms if needed

### Conclusion

For self-hosted Mattermost running on Linux AMD64, single-platform bundle is the optimal choice. It provides the same functionality with significantly better performance and smaller footprint.

---

**Related Documents:**
- [RELEASE.md](../RELEASE.md) - Release process
- [AUTO-UPDATE-GUIDE.md](AUTO-UPDATE-GUIDE.md) - Auto-update setup
- [CHANGELOG-MULTI-PLATFORM.md](CHANGELOG-MULTI-PLATFORM.md) - Previous multi-platform changes

