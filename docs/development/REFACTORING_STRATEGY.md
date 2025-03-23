# Refactoring Strategy for OpenTelemetry API Compatibility

## Current Issues

1. **API Version Mismatch**: 
   - The full implementation (with `fullwasm` build tag) uses the older OpenTelemetry API (`go.opentelemetry.io/collector/pdata`)
   - The stub implementation (with `\!fullwasm` build tag) uses the newer OpenTelemetry API (`go.opentelemetry.io/collector/pdata/ptrace`, etc.)

2. **Conflicting Utility Files**:
   - `pkg/common/utils.go` doesn't have proper build tags
   - `pkg/common/utils_pdata.go` and `pkg/common/utils_pcommon.go` have appropriate build tags but aren't importing the correct packages

3. **Failing Compilation with `fullwasm` Tag**:
   - When trying to build with the `fullwasm` tag, we get imports errors and missing wasmer-go dependencies

## Recommended Approach

### Option 1: Maintain Dual API Support (More Complex but Backward Compatible)

1. **Clearly Separate Implementations**:
   - `pkg/common/utils.go` should be removed - its functionality is duplicated
   - `pkg/common/utils_pdata.go` should only contain imports from the older API
   - `pkg/common/utils_pcommon.go` should only contain imports from the newer API
   - Both files should have appropriate build tags

2. **Properly Import Dependencies**:
   - `utils_pdata.go` should import `"go.opentelemetry.io/collector/pdata"`
   - `utils_pcommon.go` should import specific modules like `"go.opentelemetry.io/collector/pdata/ptrace"`, `"go.opentelemetry.io/collector/pdata/pcommon"`, etc.

3. **Wasmer Integration**:
   - Ensure wasmer-go is properly vendored or included as a dependency
   - Update go.mod to point to a compatible version of wasmer-go
   - Consider using a conditional build to avoid wasmer-go dependency when not needed

### Option 2: Standardize on the Newer API (Simpler but Requires Refactoring)

1. **Migrate the Full Implementation**:
   - Update `traces.go` to use the newer API (`pdata/ptrace`, etc.)
   - Remove the older utility files and just use the newer implementations
   - This will require reviewing and updating all function signatures and type references

2. **Rename and Consolidate Files**:
   - Rename `utils_pcommon.go` to something more generic (perhaps just `utils.go`)
   - Remove duplicate functionality across files

## Next Steps

1. **Decide on an Approach**: 
   - Option 1 maintains backward compatibility but increases code complexity
   - Option 2 simplifies the codebase but requires more immediate refactoring work

2. **Create a Comprehensive Test Plan**:
   - Ensure changes don't break either implementation
   - Verify both the full and stub implementations work correctly
   - Automate build testing with both build tag configurations

3. **Implement in Phases**:
   - Start with the most critical components first
   - Ensure each phase leaves the code in a buildable state
   - Add comprehensive comments to explain the dual API approach if taking Option 1

4. **Update Documentation**:
   - Document the build tag system clearly
   - Provide examples of how to build each version
   - Explain when to use each implementation
