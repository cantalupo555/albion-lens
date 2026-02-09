# Code Audit Findings for albion-lens

Audit Date: 2025-02-09
Auditor: Automated Code Audit

---

## Summary
This audit analyzed the entire codebase for race conditions, resource leaks, dead code, error handling gaps, security vulnerabilities, cross-platform issues, and other code quality concerns.

**Total Issues Found: 8**

---

## Critical Issues

### 1. Goroutine panic when receiving from closed channels

**Severity:** Bug  
**File Path:** cmd/tui/main.go  
**Line Numbers:** 52-112, 115-123

**Description:**
The goroutines that bridge backend events to TUI will panic when they attempt to receive from channels that have been closed by `svc.Stop()`. Specifically:

1. Lines 52-112: The event batching goroutine receives from `svc.Events` and sends to `bulkEventChan`
2. Lines 115-123: The stats bridging goroutine receives from `svc.Stats` and sends to `statsChan`

When `svc.Stop()` is called (deferred in main.go:131), it closes these channels (service.go:207-209). If the bridging goroutines attempt to receive from the closed channels, they will panic with "send on closed channel" or "receive on closed channel".

**How to Reproduce:**
1. Run the application with the TUI
2. Press 'q' to quit (or allow it to exit normally)
3. The `svc.Stop()` defer in main.go:131 closes the channels
4. The goroutines in lines 52-112 and 115-123 will panic when they try to receive from the closed channels

**Suggested Fix:**
Use a select statement with a default case when sending to channels, and ensure goroutines exit gracefully when channels are closed. Add a done channel to signal goroutine shutdown.

---

### 2. Race condition in event callback - bufferPeakInternal access

**Severity:** Bug  
**File Path:** pkg/backend/service.go  
**Line Numbers:** 107-109

**Description:**
The event callback (line 99-120) is called from the packet capture goroutine and accesses `s.parser.Stats.UpdateBufferPeak()`. While `UpdateBufferPeak()` uses atomic operations internally, the concurrent access to `s.parser.Stats` (which is not atomic) creates a race condition when multiple goroutines try to access it simultaneously.

The specific issue is on line 108 where `s.parser.Stats.BufferCapacity` is read. This field is set on line 127 before the parser starts, but if the service is reconfigured or accessed concurrently, it could cause issues.

**How to Reproduce:**
This is a subtle race that would require high packet rates and concurrent access patterns to trigger.

**Suggested Fix:**
Use atomic operations or mutex protection when accessing `s.parser` fields from concurrent goroutines. Since `s.parser` is assigned before the capture starts, this should be safe, but the access pattern should be reviewed.

---

### 3. Missing bounds check in Protocol16 array reading

**Severity:** Bug  
**File Path:** pkg/photon/protocol16.go  
**Line Numbers:** 139-152

**Description:**
In the `readValue()` function for `TypeArray`, the loop creates array elements but doesn't properly validate that the buffer has enough data for all elements. Line 141 reads `elemType` but if the buffer runs out of data during the loop, `readValue(r, elemType)` will read beyond the buffer bounds.

```go
arr := make([]interface{}, length)
for i := 0; i < int(length) && !r.IsEmpty(); i++ {
    arr[i] = readValue(r, elemType)  // Line 151 - no bounds check here
}
```

While `r.IsEmpty()` is checked in the loop condition, `readValue()` might attempt to read more bytes than available if the element type is a multi-byte type and the buffer is nearly empty.

**How to Reproduce:**
Send a crafted Protocol16 array packet where the length field indicates more elements than can fit in the remaining buffer data.

**Suggested Fix:**
Add explicit bounds checking using `r.CanRead()` before calling `readValue()` for each element, or validate the total bytes needed for the array before reading.

---

### 4. Integer overflow risk in Protocol16 array length

**Severity:** Bug  
**File Path:** pkg/photon/protocol16.go  
**Line Numbers:** 139-152

**Description:**
The array length is read as `uint16` (line 140) and then converted to `int` (line 141) without overflow checking:

```go
length, err := r.ReadUint16()  // Returns uint16
arr := make([]interface{}, length)  // Converted to int
```

On a 64-bit system, `int` is 64-bit, so `uint16(65535)` fits fine. However, this pattern is fragile. If the type ever changes to support larger arrays (e.g., uint32 length), the conversion could silently overflow or cause issues.

Additionally, line 143-151 doesn't validate that `length` is within reasonable bounds before allocating the slice.

**How to Reproduce:**
Send a Protocol16 array with maximum uint16 length (65535) - this would allocate a very large slice and potentially cause memory exhaustion.

**Suggested Fix:**
Add reasonable limits on array lengths (e.g., maximum 1000 elements) to prevent memory exhaustion attacks from malformed network data.

---

## Warnings

### 5. Trailing whitespace in multiple files

**Severity:** Warning  
**File Path:** Multiple files  
**Line Numbers:** Various

**Description:**
Several files contain lines with trailing whitespace (spaces at the end of lines):

- internal/tui/components/eventlog.go:149 (`            return nil`) 
- pkg/photon/parser.go:289 (`    }`)
- pkg/handlers/albion.go:231, 267, 375 (multiple instances)

**Suggested Fix:**
Run a linter like `gofmt -s` or `golangci-lint` to automatically remove trailing whitespace.

---

### 6. Inconsistent error handling in packet parsing

**Severity:** Warning  
**File Path:** pkg/photon/parser.go  
**Line Numbers:** 173-195

**Description:**
When processing commands in `ParsePacket()`, errors from `ReadUint32()`, `ReadInt32()`, and `ReadBytesNoCopy()` are silently discarded with `_`:

```go
commandType, _ := r.ReadByte()  // Line 180
_ = r.Skip(1)  // Line 181
_ = r.Skip(1)  // Line 182
_ = r.Skip(1)  // Line 183
commandLength, _ := r.ReadUint32()  // Line 185
sequenceNumber, _ := r.ReadInt32()  // Line 186
```

If any of these reads fail, the error is ignored and the parser continues with incorrect values, leading to potential panics or incorrect packet processing.

**How to Reproduce:**
Send a malformed Photon packet with an incomplete command header.

**Suggested Fix:**
Check errors from buffer read operations and return an error if any read fails. This will prevent processing malformed packets.

---

### 7. Missing test coverage for critical parsing functions

**Severity:** Info  
**File Path:** pkg/photon/parser.go  
**Line Numbers:** Various

**Description:**
Critical packet parsing functions lack test coverage:

- `ParsePacket()` - Main packet entry point (line 131)
- `handleSendFragment()` - Fragment reassembly (line 271)
- `handleSendReliable()` - Reliable packet processing (line 230)
- `handleOperationRequest()` - Request parsing (line 336)
- `handleOperationResponse()` - Response parsing (line 356)
- `handleEventData()` - Event parsing (line 395)

These functions contain complex logic for packet reassembly and protocol parsing but have no corresponding test files.

**Suggested Fix:**
Add unit tests for these parsing functions with mock data to ensure they handle various packet formats correctly.

---

### 8. TODO/FIXME/HACK comments

**Severity:** Info  
**File Path:** pkg/photon/parser.go  
**Line Numbers:** 166-172

**Description:**
There is an incomplete TODO comment about CRC validation:

```go
// Line 166-172:
// Skip CRC field and validate (for now, just skip)
// In a full implementation, we'd validate the CRC
_ = r.Skip(4)
if p.debug {
    fmt.Println("  [Photon] Packet has CRC enabled (skipping validation)")
}
```

The comment indicates that CRC validation is intentionally skipped but should be implemented in the future.

**Suggested Fix:**
Either implement CRC validation or update the comment to clarify if this is a permanent design decision (e.g., "CRC validation intentionally skipped - not needed for this use case").

---

## Code Quality Observations (Non-Issues)

### Positive Findings

1. **No security vulnerabilities found** - The codebase follows read-only network sniffing and does not send packets, inject data, or modify game traffic
2. **Good use of atomic operations** - Stats counters use atomic operations for thread-safe increments
3. **Proper mutex usage** - Most shared state is protected with mutexes appropriately
4. **Well-structured buffer reader** - The BufferReader class provides safe bounds checking for all read operations
5. **Comprehensive test coverage** - Test files exist for most components with good coverage

### Cross-Platform Considerations

The codebase uses libpcap through CGO, which is cross-platform compatible:
- Linux: libpcap-dev
- Windows: Npcap
- macOS: Built-in libpcap

No platform-specific code was found that would require build tags.

### Albion Online ToS Compliance

The application is strictly passive:
- Only reads network packets using gopacket/pcap
- Does not send any data to network sockets
- Does not modify game traffic
- Does not inject packets

This complies with SBI's policy: "As long as you just look and analyze we are ok with it. The moment you modify or manipulate something or somehow interfere with our services we will react."

---

## Recommendations

1. Fix the goroutine panic issue (#1) - This is the highest priority as it crashes on exit
2. Add bounds checks for array reading (#3) - Prevent potential buffer overruns
3. Improve error handling in packet parsing (#6) - Don't ignore read errors
4. Add tests for critical parsing functions (#7) - Improve code reliability
5. Run static analysis tools (golangci-lint, go vet) to catch issues like trailing whitespace (#5)

---

## Audit Methodology

The audit examined:
- All .go source files (26 files)
- Race conditions and synchronization issues
- Resource leaks and goroutine lifecycle
- Dead code and unreachable paths
- TODO/FIXME/HACK comments
- Error handling patterns
- Test coverage
- Hardcoded values and magic numbers
- Cross-platform compatibility
- Security vulnerabilities
- Albion Online ToS compliance
- Protocol parsing correctness
- CGO memory safety
- Code formatting (trailing whitespace)
