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
- Each trace line now includes the program counter, the opcode byte, and the assembled instruction. Register and flag state are shown in the static trace status panel.

## Video Backends

- Use `-video-backend null` for the current no-op renderer.
- Use `-video-backend vulkan` with a binary built using `make build-vulkan` to open a basic GLFW window backed by a Vulkan instance and surface.
- If Vulkan runtime initialization fails on the host, startup fails with the underlying Vulkan error.
- The current Vulkan backend stops at window, instance, and surface bring-up; the swapchain upload path and CRT shader pipeline are still pending.
- On macOS, `make build-vulkan` auto-detects MoltenVK in `VULKAN_SDK`, `~/VulkanSDK/latest/macOS/lib`, `/usr/local/lib`, and `/opt/homebrew/lib`, then injects the required runtime search paths into the binary.
- Example: `make build-vulkan && ./bin/emuai -video-backend vulkan`.
- Tune the base framebuffer with `-video-width`, `-video-height`, and `-video-refresh-hz`.
- The video package now contains a CRT-oriented configuration model, a framebuffer snapshot pipeline, and a renderer interface ready to host a Vulkan pass chain for phosphor persistence, scanlines, mask simulation, and curvature.

## Running A Binary

- Use `go run ./cmd/emuai -bin assets/6502_functional_test.bin -load-addr 0x0400` to load a raw binary into RAM and set the reset vector to that address.
- The `-pc` option is deprecated and ignored at launch; startup always uses the CPU reset vector at `$FFFC`.
- Use `-stop-pc` to stop cleanly before executing the instruction at a specific program counter value.
- Example: `go run ./cmd/emuai -bin assets/6502_functional_test.bin -load-addr 0x0400`.
- Example with success trap: `./bin/emuai -bin assets/6502_functional_test.bin -load-addr 0x0400 -stop-pc 0x3469`.
- Use `-max-cycles` to stop after a specific number of motherboard cycles.
- Use `-timeout` to cap wall-clock runtime, or leave it at `0` to run until halt, cycle limit, or manual interruption.
- Use `-realtime` if you explicitly want clock-driven execution; by default the CLI steps as fast as possible.
- External binaries loaded with `-bin` use real `BRK` interrupt behavior instead of stopping the emulator on `BRK`.
- If neither `-bin` nor `-rom-config` is provided, the CLI boots the default Apple II ROM set from `ROMs/apple2-roms.yaml`.
- The machine state is cleared first, then boot sources are loaded, and only then is the CPU reset triggered so the reset vector sees the final ROM/RAM contents.

## Loading ROMs From YAML

- Use `-rom-config` to load one or more ROM images into RAM from a YAML file.
- Each ROM entry defines a `path` and a `start` address; paths are resolved relative to the YAML file.
- Example: `go run ./cmd/emuai -rom-config ROMs/apple2-roms.yaml`.
- If `-rom-config` is omitted, the default Apple II ROM config at `ROMs/apple2-roms.yaml` is used automatically.

Example configuration:

```yaml
roms:
  - name: apple2-system-bank-d
    path: D.bin
    start: 0xD000
  - name: apple2-system-bank-ef
    path: EF.bin
    start: 0xE000
```

  ## Pause Menu And Memory Dump

  - Press space while the emulator is running to open the pause menu.
  - In the pause menu, use `r` to resume, `m` to dump memory as 6502 assembly, and `q` to stop execution.
  - The memory dump prompts for a start address and prints 16 decoded instructions using the same byte and assembly view as the CPU trace.
  - Accepted address formats are `0xD000`, `$D000`, or `D000`.

## Next Steps

- Add a real 6502 or 6510 core in `internal/components/cpu`.
- Add ROM loading helpers and machine profiles.
- Add VIC-II / SID style chips as memory-mapped devices.
- Add deterministic tests for cycle-accurate behavior.
