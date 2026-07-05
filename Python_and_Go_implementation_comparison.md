# Leading Edge — Python vs. Go implementation comparison

This document analyses how the Go port in this folder relates to the original
`leadingedge.py`. It covers the structural mapping, the language‑paradigm
differences that shaped the port, the framework substitutions, and a set of
subtle numeric/semantic details that had to be reproduced exactly for the game
to behave the same way.

The goal throughout the port was **behavioural fidelity**: the Go code is a
faithful line‑by‑line translation of the game logic, deviating only where a
language or library difference forces a different expression of the same idea.

---

## 1. High‑level architecture

Both versions share the same conceptual design:

- A **pseudo‑3D racer**. The track is a list of cross‑section "pieces", each
  with X/Y offsets from the previous one. A perspective transform projects them
  to screen space; polygons connect consecutive pieces; sprites (cars, scenery)
  are scaled by distance.
- A **painter's algorithm**: draw calls are collected into a list and executed
  back‑to‑front so distant geometry is drawn first.
- A **fixed 1/60 s timestep** decoupled from the render frame rate via an
  accumulator.
- A **title/demo → play → game‑over** state machine, where the title screen
  runs a self‑playing demo race.

The largest single piece of logic in both is the track renderer inside
`Game.draw` / `(*Game).Draw`, which is ported statement‑for‑statement.

---

## 2. Language paradigm: classes/inheritance → interfaces/embedding

This is the biggest structural difference.

### Python: classical inheritance with a polymorphic list

```python
class Car:                       # base
    def update(self, delta_time): ...
class CPUCar(Car):               # subclass
    def update(self, delta_time):
        ...
        super().update(delta_time)
class PlayerCar(Car):            # subclass
    ...

self.cars = [PlayerCar(...), CPUCar(...), ...]   # heterogeneous list
for car in self.cars:
    car.update(delta_time)       # dynamic dispatch
```

`game.cars` is a single list holding both car kinds, iterated polymorphically.
Collision code even reaches into subclass‑specific fields (`car.target_speed`)
without checking the type, relying on the fact that the player only ever
collides with CPU cars.

### Go: interface + struct embedding

Go has no inheritance, so the port uses an **interface** for polymorphism and
**struct embedding** for code reuse:

```go
type Car interface {
    Base() *BaseCar
    Update(g *Game, dt float64)
    UpdateCurrentTrackPiece(g *Game)
    IsCPU() bool
}

type BaseCar struct {            // "base class" fields
    self         Car             // back-reference (see below)
    Pos          Vec3
    Speed        float64
    ...
}

type CPUCar struct {
    BaseCar                      // embedding == "extends"
    TargetSpeed float64
    ...
}
type PlayerCar struct {
    BaseCar
    ...
}
```

`super().update()` becomes an explicit call to an embedded method:

```go
func (c *CPUCar) Update(g *Game, dt float64) {
    ...
    c.baseUpdate(g, dt)          // was: super().update(delta_time)
}
```

Two consequences worth noting:

1. **The `self` back‑reference.** In Python, `self` is always the concrete
   object, so `track_piece.cars.remove(self)` and `car is self` work naturally.
   In Go, a method on the embedded `BaseCar` doesn't know which outer struct
   (`*CPUCar`/`*PlayerCar`) contains it. The port stores the wrapping interface
   value in `BaseCar.self` at construction (`c.self = c`) so shared code can do
   identity comparisons and list removal:

   ```go
   b.trackPiece.Cars = removeCar(b.trackPiece.Cars, b.self)
   ```

2. **Reaching into subclass fields** (`car.target_speed = car.speed`) becomes a
   type assertion instead of duck typing:

   ```go
   if cpu, ok := other.(*CPUCar); ok {
       cpu.TargetSpeed = ob.Speed
   }
   ```

---

## 3. File organisation

Python keeps everything in one 1,664‑line module. Go favours many small files
in one `package main`, so the port is split by concern:

| Python (`leadingedge.py` sections) | Go file |
|---|---|
| `update()`, `draw()`, state machine, mixer setup | `main.go` |
| `Game` class (update, draw/render, track lookups) | `game.go` |
| `TrackPiece`, `TrackPieceStartLine`, `make_track` | `track.go` |
| `Scenery`, `StartGantry`, `Billboard`, `Lamp*` | `scenery.go` |
| `Car`, `update_sprite`, `update_current_track_piece` | `car.go` |
| `CPUCar` | `cpucar.go` |
| `PlayerCar` | `playercar.go` |
| image/text helpers, `draw_text`, `text_width` | `assets.go` |
| sound/music (`play_sound`, engine/skid handling) | `audio.go` |
| `Controls`, `KeyboardControls` | `input.go` |
| `remap`, `sign`, `move_towards`, `format_time`, vectors | `util.go` |
| module‑level constants | `constants.go` |
| `randint`/`uniform`/`choice` wrappers | `rng.go` |

---

## 4. Framework: Pygame Zero → go‑sdl3

| Concern | Python (Pygame Zero / Pygame) | Go (go‑sdl3) |
|---|---|---|
| Window / loop | `pgzrun.go()` calls `update`/`draw` | explicit `sdl.RunLoop` with manual clear/present |
| Filled polygons | `pygame.draw.polygon(surface, col, points)` | `renderer.RenderGeometry` with a triangle fan |
| Sprite scaling | `pygame.transform.scale(img, (w,h))` then `blit` | `renderer.RenderTexture(tex, nil, dstRect)` (dst W/H scales) |
| Images | `images.foo` attribute access, auto‑loaded | lazy `Assets.Texture(name)` cache of `*sdl.Texture` |
| Text | sprite‑font blitting via `getattr(images, ...)` | same technique, names built with `strconv.Itoa` |
| Alpha overlay | `Surface.set_alpha` + `blit` | `SetDrawBlendMode(BLEND)` + `RenderFillRect` |
| Sound effects | `sounds.foo.play()` | preloaded `map[string]*mixer.Audio` + `PlayAudio` |
| Looping sound | `sound.play(loops=-1, fade_ms=…)` | a `*mixer.Track` with `SetLoops(-1)` + `Play(ms)` |
| Music | `music.play(name)` | dedicated looping `*mixer.Track` per track |

### Filled polygons

Pygame fills an arbitrary polygon directly. SDL3 only draws triangles, so
`Assets.FillPolygon` triangulates each (convex) quad as a fan and submits
vertex‑coloured geometry with no texture:

```go
func (a *Assets) FillPolygon(points []Vec2, c RGB) {
    col := sdl.FColor{R: float32(c.R) / 255, ...}
    verts := make([]sdl.Vertex, len(points))
    for i, p := range points { verts[i] = sdl.Vertex{Position: ..., Color: col} }
    indices := make([]int32, 0, (len(points)-2)*3)
    for i := 1; i < len(points)-1; i++ {
        indices = append(indices, 0, int32(i), int32(i+1))
    }
    a.renderer.RenderGeometry(nil, verts, indices)
}
```

### Sprite scaling

Python pre‑scales a surface (`SCALE_FUNC`, nearest‑neighbour) and blits it; Go
lets the renderer scale during blit by supplying a destination rect and forces
nearest‑neighbour filtering to match:

```go
tex.SetScaleMode(sdl.SCALEMODE_NEAREST)   // mimic pygame.transform.scale
```

### The draw list (painter's algorithm)

Python appends `lambda`s to `draw_list`, then runs `reversed(draw_list)`. Go
appends closures to `[]func()` and iterates the slice backwards:

```go
var drawList []func()
add := func(fn func()) { drawList = append(drawList, fn) }
...
for k := len(drawList) - 1; k >= 0; k-- { drawList[k]() }
```

Go 1.22+ per‑iteration loop variable semantics make capturing values in these
closures safe; the port additionally copies polygon point slices before
capturing them to avoid aliasing the reused locals.

---

## 5. Vectors and math

Pygame supplies `Vector2`/`Vector3` with operator overloading. Go has no
operator overloading, so `util.go` defines value‑type vectors with methods:

```go
type Vec3 struct{ X, Y, Z float64 }
func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Scale(s float64) Vec3 { ... }
```

So Python `offset += offset_delta` becomes `offset = offset.Add(offsetDelta)`,
and `Vector2(px, py) * fraction` becomes `Vec2{px, py}.Scale(fraction)`.

Using **value types** (not pointers) reproduces pygame's copy semantics for
free: `car_offset = Vector3(offset)` (an explicit copy in Python) is just
`carOffset := offset` in Go.

### Numeric semantics that had to be matched exactly

These are the easy‑to‑miss details where Python and Go differ by default:

- **Integer vs float division.** Python `//` is floor division; `/` is float.
  The renderer's stripe/rumble/trackside colour selection uses `i // 3 % 2`,
  ported as Go integer division `(i/3)%2` (Go `/` on `int` truncates, which
  matches here because `i ≥ 0`).

- **`int()` truncation vs `math.Floor`.** The two track lookups deliberately
  differ in the original and are preserved:

  ```python
  def get_track_piece_for_z(self, z):      idx = -int(z / SPACING)          # truncate toward zero
  def get_first_track_piece_ahead(self, z):idx = -int(math.floor(z/SPACING))# floor
  ```
  ```go
  func (g *Game) getTrackPieceForZ(z float64)      (int, bool) { idx := -int(z / Spacing); ... }
  func (g *Game) getFirstTrackPieceAhead(z float64) (int, float64, bool) { idx := -int(math.Floor(z/Spacing)); ... }
  ```
  Go's `int(float)` conversion truncates toward zero, matching Python's `int()`.

- **Power operator.** Python `drag_factor ** (delta_time/(1/60))` becomes
  `math.Pow(dragFactor, dt/(1.0/60.0))`.

- **`math.Mod` for float modulo.** `car.pos.z % SPACING != 0` →
  `math.Mod(cb.Pos.Z, Spacing) != 0`; `int(self.tyre_rotation % 2)` →
  `int(math.Mod(b.TyreRotation, 2))`.

- **String formatting.** `f"{seconds%60:06.3f}"` → `fmt.Sprintf("%06.3f", rem)`;
  `f"{self.player_car.lap:02}"` → `fmt.Sprintf("%02d", pc.lap)`;
  `f"explode{t//2:02}"` → `fmt.Sprintf("explode%02d", p.explodeTimer/2)`.

- **Builtin `min`/`max`.** Go 1.21+ provides generic `min`/`max`, so
  `min(max(angle_sprite_idx, -1), 1)` ports verbatim as
  `min(max(angleIdx, -1), 1)`.

---

## 6. Dynamic typing → static typing

### `None` → pointers or companion booleans

Python uses `None` for "no value" on many fields. Go has no `None`, so the port
uses either a nil pointer or a value + `has*` boolean, chosen for clarity:

| Python | Go |
|---|---|
| `self.explode_timer = None / int` | `explodeTimer int` + `exploding bool` |
| `self.last_checkpoint_idx = None / int` | `lastCheckpointIdx int` + `hasCheckpoint bool` |
| `self.fastest_lap = None / float` | `fastestLap float64` + `hasFastest bool` |
| `cpu_max_target_speed = None / number` | `CPUMaxTargetSpeed float64` + `HasCPUMax bool` |
| `get_first_track_piece_ahead → None,None` | returns `(int, float64, bool)` |

For example:

```python
if self.fastest_lap is None or self.lap_time < self.fastest_lap:
    self.fastest_lap = self.lap_time
```
```go
if !p.hasFastest || p.lapTime < p.fastestLap {
    p.fastestLap = p.lapTime
    p.hasFastest = true
}
```

### `getattr(images, name)` → a map / string names

Python builds sprite names as strings and resolves them dynamically:

```python
image = getattr(images, font + "0" + str(ord(char)))
```

Go keeps images as string keys into a texture cache, so the equivalent is just
string construction plus a map lookup inside `Assets.Texture`:

```go
name := font + "0" + strconv.Itoa(int(char))
```

The special controller‑button glyph (`SPECIAL_FONT_SYMBOLS = {'xb_a':'%'}`) is
handled inline: when the character is `'%'`, the sprite name becomes `"xb_a"`.

### Ad‑hoc classes → typed structs

`Scenery` and its subclasses (`StartGantry`, `Billboard`, `LampLeft/Right`)
collapse into a single `Scenery` struct with constructor functions, because the
only behavioural override is the start gantry's animated image. That override
becomes a boolean flag consulted in one method:

```go
func (s *Scenery) GetImage(g *Game) string {
    if s.isStartGantry { ... s.Image = "start" + strconv.Itoa(index) }
    return s.Image
}
```

---

## 7. Nullable / optional handling and defensive guards

Python occasionally relies on conditions that *would* raise if a value were
`None`/out of range, but in practice never do (e.g. negative list indexing when
the camera is before the track start, or `prev_ahead >= 0` when `prev_ahead`
could be `None`). Because Go panics on out‑of‑range slice access rather than
silently wrapping (Python's negative indexing), the port adds small guards that
are logically no‑ops in normal play but prevent crashes at the edges:

```go
if firstIdx < 0 {          // camera before track start (shouldn't happen)
    firstIdx = 0
    currentPieceZ = 0
}
...
if math.Mod(cb.Pos.Z, Spacing) != 0 && i+1 < len(g.track) {   // guard track[i+1]
```

These are the only places the Go logic is *stricter* than the Python, and they
never change observable behaviour during a real race.

---

## 8. Audio

The sound model is the same, but the plumbing differs. Both:

- play random one‑shot variants (`play_sound("bump", 6)` picks `bump0`..`bump5`);
- keep one **looping engine sound** whose sample is chosen by speed
  (`min(int(speed*0.6), 39)`), switching sample when the index changes;
- keep one **looping skid sound** whose volume tracks grip;
- treat the stereo `ambience` and the menu themes as **music**.

Python calls methods on `Sound` objects directly (`sound.play(loops=-1,
fade_ms=100)`, `sound.set_volume(v)`, `sound.fadeout(250)`). go‑sdl3 models a
persistent **`Track`** you re‑point at different audio, so the port keeps a
dedicated engine track and skid track and mutates them:

```go
func (a *Audio) UpdateEngineSound(speed float64) {
    idx := clamp(int(speed*0.6), 0, len-1)
    if idx != a.currentEngineIdx {
        a.currentEngineIdx = idx
        a.engineTrack.SetAudio(a.engineSounds[idx])
        a.engineTrack.SetGain(0.3)
        a.engineTrack.Play(100)
    }
}
```

All audio is best‑effort with nil checks, mirroring the Python `try/except`
blocks that let the game run without sound hardware.

**Intentional simplification:** the Python `play_sound` supports per‑instance
volume scaling for a few effects; the current go‑sdl3 mixer wrapper plays
one‑shots at their default gain. This affects only the loudness of some effects,
not gameplay.

---

## 9. Input

Python's `Controls` is an abstract base with keyboard and joystick subclasses,
and it tracks button edges (`is_button_pressed`) in an `update()` method called
each frame. The Go port:

- implements **keyboard only** (no joystick), as a stateless `Controls{}` value;
- reads a per‑frame snapshot of the SDL keyboard state (`keys`/`prevKeys`) and
  derives held vs. just‑pressed from it:

  ```go
  func keyJustPressed(sc sdl.Scancode) bool {
      return keys[sc] && (prevKeys == nil || !prevKeys[sc])
  }
  ```

This reproduces the edge‑detection used for "press to start / restart" while
sidestepping the double‑`update()` quirk in the original (where `PlayerCar`
calls `self.controls.update()` again mid‑frame). Held buttons (accelerate/brake)
and the steering axis behave identically.

---

## 10. Game loop and timing

Both use the same fixed‑timestep accumulator. Pygame Zero hands `update` a
`delta_time`; the Go loop computes it from `sdl.Ticks()`:

```go
now := sdl.Ticks()
dt := float64(now-lastTicks) / 1000.0
lastTicks = now
if dt > 0.1 { dt = 0.1 }        // clamp to avoid a huge catch-up burst
...
for accumulatedTime >= FixedTimestep {
    accumulatedTime -= FixedTimestep
    game.Update(FixedTimestep)
}
```

The state machine matches, including the subtlety that switching Title→Play
creates a new `Game` *before* the accumulator loop runs, so the freshly created
game gets updated on the same frame.

---

## 11. Idiom translation cheat‑sheet

| Python idiom | Go equivalent used |
|---|---|
| list comprehension building track pieces | `add(n, func(i int) *TrackPiece {...})` helper loop |
| `self.cars.sort(key=lambda c: c.pos.z)` | `sort.SliceStable(g.cars, func(i,j int) bool {...})` |
| `cars_to_draw.sort(key=…, reverse=True)` | `sort.SliceStable(..., a.z > b.z)` |
| `game.cars.index(self)` | `g.indexOfCar(p.self)` linear scan |
| `[x for x in xs if cond]` | explicit `for` + `append` |
| `random.uniform(a,b)` / `choice(...)` / `randint` | `uniform`, `choiceStr`, `randIntn` in `rng.go` |
| `Vector3(offset)` (copy) | `carOffset := offset` (value copy) |
| `str(index)` | `strconv.Itoa(index)` |
| f‑strings | `fmt.Sprintf` |
| `try/except` around audio | nil checks / best‑effort no‑ops |

---

## 12. What is intentionally identical

- Track layout (`make_track`): every section, offset, length, scenery choice,
  interval and CPU speed cap is reproduced piece‑for‑piece.
- The perspective transform, clipping planes, and all polygon vertex orderings
  (stripe, yellow line, track, rumble, trackside) — including the exact tuple
  index gymnastics used to build the trackside quads.
- Camera follow logic and the background‑offset accumulation across track‑piece
  boundaries (single‑piece, two‑piece, and multi‑piece movement cases).
- Player physics: drag, grip loss on corners, steering strength, the corner
  "offset" illusion, collisions (side/front/back), grass drag, reset/explode,
  checkpoints, lap and fastest‑lap timing.
- CPU AI: target‑speed drift, corner speed caps, target‑X selection avoiding
  nearby cars, and the angle‑sprite selection based on apparent bearing.
- UI: status bar layout, "FASTEST LAP!/FINAL LAP!/RACE COMPLETE!/TIME UP!"
  timing, and the title‑screen fade‑in/out.

---

## 13. Summary of differences

| Category | Difference | Reason |
|---|---|---|
| Paradigm | inheritance → interface + embedding + `self` back‑ref | Go has no classes |
| Optionals | `None` → nil pointers / `has*` bools | Go has no `None` |
| Dynamic names | `getattr` → string keys + map cache | static typing |
| Vectors | operators → methods, value semantics | no operator overloading |
| Polygons | `draw.polygon` → triangulated `RenderGeometry` | SDL draws triangles only |
| Scaling | pre‑scale surface → dest‑rect blit + nearest mode | renderer scales on blit |
| Audio | `Sound` methods → persistent `Track`s | go‑sdl3 mixer model |
| Input | keyboard+joystick classes → stateless keyboard snapshot | scope / simplicity |
| Safety | negative indexing/`None` compares → explicit guards | Go panics instead of wrapping |
| Simplifications | per‑sound volume scaling; joystick support | not yet ported |

Overall the Go version is a close structural mirror of the Python: the control
flow, constants, and math are line‑by‑line equivalent, and the differences are
almost entirely mechanical consequences of static typing, the absence of
inheritance/operator overloading, and swapping Pygame Zero for go‑sdl3.
