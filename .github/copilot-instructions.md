# SkipTrie-Go Development Instructions

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Repository Overview
SkipTrie-Go is a Go implementation of a SkipTrie data structure - a probabilistic data structure that combines features of skip lists and tries for efficient predecessor queries on integer keys. The codebase is minimal and focused, consisting of:

- **skiptrie/skiptrie.go**: Main SkipTrie implementation with Insert, Contains, Predecessor, and Delete operations
- **skiptrie/skiptrie_test.go**: Comprehensive test suite with unit tests and benchmarks  
- **simple_demo.go**: Basic demonstration of SkipTrie functionality
- **go.mod**: Go module configuration (Go 1.24.6)
- **README.md**: Basic repository description

## Working Effectively

### Required Dependencies
- **Go 1.24.6 or later** - Install from https://golang.org/dl/
- No external dependencies required - uses only Go standard library

### Bootstrap and Build Commands
Execute commands from the repository root (`/path/to/skiptrie-go`):

1. **Initialize/Verify Module**: `go mod tidy`
   - Takes ~0.1 seconds
   - Safe to run multiple times

2. **Build All Packages**: `go build ./...`  
   - Takes ~0.1 seconds - NEVER CANCEL, set timeout to 30+ seconds
   - Compiles all Go packages in the repository
   - Must pass before running tests or demos

3. **Format Code**: `go fmt ./...`
   - Takes ~0.1 seconds  
   - Automatically formats all Go files to standard style
   - ALWAYS run before committing changes

4. **Static Analysis**: `go vet ./...`
   - Takes ~0.1 seconds
   - Checks for common Go programming errors
   - Must pass with no warnings

### Testing Commands
1. **Run All Tests**: `go test ./...`
   - Takes ~0.2 seconds - NEVER CANCEL, set timeout to 60+ seconds
   - Note: Some tests (TestDelete, TestRandomOperations) currently fail due to incomplete delete implementation
   - Core functionality tests (TestInsertAndContains, TestNewSkipTrie, TestInsertDuplicate) pass consistently

2. **Run Core Tests Only**: `go test -run "TestInsertAndContains|TestNewSkipTrie|TestInsertDuplicate" ./...`
   - Takes ~0.15 seconds - NEVER CANCEL, set timeout to 30+ seconds
   - Runs only the stable, working tests
   - Use this for validating core functionality changes

3. **Verbose Testing**: `go test -v ./...`
   - Takes ~0.2 seconds - NEVER CANCEL, set timeout to 60+ seconds
   - Shows individual test results and timing

4. **Benchmarks**: `go test -bench=. ./...`
   - Takes ~0.2 seconds - NEVER CANCEL, set timeout to 60+ seconds  
   - Runs performance benchmarks for Insert and Contains operations

### Running the Application
1. **Demo Application**: `go run simple_demo.go`
   - Takes ~0.1 seconds - NEVER CANCEL, set timeout to 30+ seconds
   - Demonstrates basic SkipTrie operations: insert, search, predecessor queries
   - Expected output shows successful insertion and retrieval of keys 10 and 20

## Validation Requirements

### Pre-Commit Validation
ALWAYS run this sequence before committing any changes:
```bash
go fmt ./...
go vet ./...
go build ./...
go test -run "TestInsertAndContains|TestNewSkipTrie|TestInsertDuplicate" ./...
```
- Total time: ~0.5 seconds - NEVER CANCEL, set timeout to 120+ seconds

### Manual Testing Scenarios
After making changes to the SkipTrie implementation:

1. **Basic Functionality Test**: `go run simple_demo.go`
   - Verify all operations show "✓" success indicators
   - Check that predecessor of 15 returns 10 when keys 10 and 20 are inserted

2. **Core API Validation**: Create a SkipTrie, insert multiple keys, verify Contains returns true for inserted keys and false for non-inserted keys

3. **Error Handling**: Test insertion of duplicate keys (should return false), test Contains on empty structure

## Known Issues and Limitations
- **Delete Implementation**: The Delete operation and related tests (TestDelete, TestRandomOperations) are currently unreliable due to incomplete implementation of node removal from skip list levels
- **Concurrent Access**: While the code uses atomic operations, full concurrent safety is not guaranteed
- **Memory Management**: Deleted nodes are marked but not physically removed, which may cause memory leaks in long-running applications

## Important Implementation Details

### Core Data Structures
- **Node**: Represents a skip list node with atomic pointers for thread safety
- **SkipTrie**: Main structure combining skip list with trie-like properties
- **LogLogU**: Maximum height is 5 levels (log log u for u = 2^32)
- **MaxKey**: Maximum key value is (2^32 - 1)

### Key Operations Performance
- **Insert**: O(log n) expected time with random height generation
- **Contains**: O(log n) expected time with level-by-level search
- **Predecessor**: O(log n) expected time using skip list traversal

### File Organization
- Keep the main implementation in `skiptrie/skiptrie.go`
- Add new tests to `skiptrie/skiptrie_test.go`
- Use `simple_demo.go` for quick functionality demonstrations
- Export only the public API methods (Insert, Contains, Predecessor, Delete, Key)

## Common Development Tasks

### Adding New Features
1. Run existing tests to ensure no regressions: `go test -run "TestInsertAndContains|TestNewSkipTrie|TestInsertDuplicate" ./...`
2. Implement new functionality in `skiptrie/skiptrie.go`
3. Add corresponding tests in `skiptrie/skiptrie_test.go`
4. Validate with format, vet, build, and test sequence
5. Test manually with demo program or custom test

### Debugging Issues
1. **Build Failures**: Check for syntax errors with `go build ./...`
2. **Test Failures**: Use `go test -v ./...` to see detailed test output
3. **Logic Issues**: Add debug prints to `simple_demo.go` and run with `go run simple_demo.go`

### Performance Analysis
1. **Benchmarking**: `go test -bench=. -benchmem ./...` shows memory allocations
2. **Profiling**: Use `go test -cpuprofile=cpu.prof -bench=. ./...` for detailed performance analysis

## Quick Reference Commands

```bash
# Repository root structure
ls -la
# Expected: .git/ README.md go.mod simple_demo.go skiptrie/

# Package contents  
ls -la skiptrie/
# Expected: skiptrie.go skiptrie_test.go

# Verify Go version
go version
# Expected: go version go1.24.6 linux/amd64 (or later)

# Check module status
go mod verify && go mod tidy
# Should complete without errors

# Full validation sequence
go fmt ./... && go vet ./... && go build ./... && go test -run "TestInsertAndContains|TestNewSkipTrie|TestInsertDuplicate" ./...
# Should complete successfully in under 1 second
```

## Repository Status
- **Buildable**: ✓ Code compiles without errors
- **Core Functionality**: ✓ Insert, Contains, and Predecessor operations work reliably  
- **Testing**: ⚠️ Basic tests pass, but Delete functionality needs improvement
- **Documentation**: ⚠️ Minimal - consider expanding README.md for public usage
- **CI/CD**: ❌ No automated workflows - consider adding GitHub Actions for continuous testing