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

## Next Steps

- Add a real 6502 or 6510 core in `internal/components/cpu`.
- Add ROM loading helpers and machine profiles.
- Add VIC-II / SID style chips as memory-mapped devices.
- Add deterministic tests for cycle-accurate behavior.
