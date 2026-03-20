# emuai

Modular 8-bit machine emulator scaffold in Go.

## Goals

- Keep the core generic so multiple machines can be assembled (Apple IIe style, C64 style, etc.).
- Run all attached components on each motherboard clock cycle.
- Isolate memory-mapped devices behind a common bus.
- Make new modules easy to plug in (CPU, RAM, video, sound, peripherals).

## Project Structure

- `cmd/emuai`: executable entrypoint and machine assembly example.
- `internal/emulator`: motherboard, clocked component contracts, and bus.
- `internal/components`: sample pluggable modules.
  - `cpu`: CPU skeleton.
  - `memory`: RAM component.
  - `video`: video stub.
  - `sound`: sound stub.
  - `peripheral`: peripheral stub.

## Core Concepts

- `Motherboard` owns:
  - the machine frequency,
  - the shared bus,
  - all clocked components,
  - the global cycle counter.
- On each tick, the motherboard calls all components in parallel and waits for completion.
- Address space is composed using `Bus.MapDevice(start, end, device)`.

## Running With Instruction Trace

- Use `go run ./cmd/emuai -trace` to print each 6502 instruction as it starts executing.
- Each trace line includes the motherboard cycle, program counter, a `FLOW` marker (`NEW` on first visit, `SEEN#n` on revisits), opcode bytes, mnemonic, CPU registers, the raw `P` value, and a decoded `flags=NVUBDIZC` view where unset flags are shown as `.`.

## Running A Binary

- Use `go run ./cmd/emuai -bin assets/6502_functional_test.bin -load-addr 0x0400` to load a raw binary into RAM and execute it from that address.
- Use `-pc` when the CPU must start from an address different from the binary load address.
- Use `-stop-pc` to stop cleanly before executing the instruction at a specific program counter value.
- Example: `go run ./cmd/emuai -bin assets/6502_functional_test.bin -load-addr 0x0400 -pc 0x040A`.
- Example with success trap: `./bin/emuai -bin assets/6502_functional_test.bin -load-addr 0x0000 -pc 0x0400 -stop-pc 0x3469`.
- Use `-max-cycles` to stop after a specific number of motherboard cycles.
- Use `-timeout` to cap wall-clock runtime, or leave it at `0` to run until halt, cycle limit, or manual interruption.
- Use `-realtime` if you explicitly want clock-driven execution; by default the CLI steps as fast as possible.
- External binaries loaded with `-bin` use real `BRK` interrupt behavior instead of stopping the emulator on `BRK`.
- If `-bin` is omitted, the built-in demo program is still loaded into RAM.
- The binary is loaded after motherboard reset so the RAM contents are preserved for execution.

## Next Steps

- Add a real 6502 or 6510 core in `internal/components/cpu`.
- Add ROM loading helpers and machine profiles.
- Add VIC-II / SID style chips as memory-mapped devices.
- Add deterministic tests for cycle-accurate behavior.
