package main

import (
	"embed"
	"path"
	"strconv"

	"github.com/Zyko0/go-sdl3/mixer"
	"github.com/Zyko0/go-sdl3/sdl"
)

// audioFS embeds the sound effects and music into the binary.
//
//go:embed sounds music
var audioFS embed.FS

// Audio wraps SDL3_mixer. All operations are best-effort so the game still runs
// without working sound hardware.
type Audio struct {
	mixer  *mixer.Mixer
	sounds map[string]*mixer.Audio

	music        map[string]*mixer.Track
	currentMusic *mixer.Track

	engineSounds     []*mixer.Audio
	engineTrack      *mixer.Track
	currentEngineIdx int

	skidTrack   *mixer.Track
	skidPlaying bool
}

func NewAudio() *Audio {
	a := &Audio{
		sounds:           make(map[string]*mixer.Audio),
		music:            make(map[string]*mixer.Track),
		currentEngineIdx: -1,
	}

	if err := mixer.Init(); err != nil {
		return a
	}
	m, err := mixer.CreateMixerDevice(sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK, nil)
	if err != nil {
		return a
	}
	a.mixer = m

	// Preload every embedded .ogg sound effect.
	entries, _ := audioFS.ReadDir("sounds")
	for _, e := range entries {
		fname := e.Name()
		if path.Ext(fname) != ".ogg" {
			continue
		}
		if snd := loadAudioFromFS(m, "sounds/"+fname); snd != nil {
			a.sounds[fname[:len(fname)-len(".ogg")]] = snd
		}
	}

	// Looping music tracks
	for _, name := range []string{"title_theme", "engines_startline", "ambience"} {
		a.music[name] = a.loopingTrack(m, "music/"+name+".ogg", 0.5)
	}

	// Engine sound bank (engine_short0 .. engine_short39)
	for i := 0; i < 40; i++ {
		a.engineSounds = append(a.engineSounds, a.sounds["engine_short"+strconv.Itoa(i)])
	}
	if track, err := m.CreateTrack(); err == nil {
		a.engineTrack = track
	}

	// Skid loop track
	if snd, ok := a.sounds["skid_loop0"]; ok {
		if track, err := m.CreateTrack(); err == nil {
			track.SetAudio(snd)
			track.SetLoops(-1)
			a.skidTrack = track
		}
	}

	return a
}

// loadAudioFromFS decodes an embedded audio file into an in-memory Audio via an
// SDL IOStream (predecoded, so no stream stays open afterwards).
func loadAudioFromFS(m *mixer.Mixer, p string) *mixer.Audio {
	data, err := audioFS.ReadFile(p)
	if err != nil {
		return nil
	}
	stream, err := sdl.IOFromConstMem(data)
	if err != nil {
		return nil
	}
	snd, err := m.LoadAudio_IO(stream, true, true) // predecode + closeio
	if err != nil {
		return nil
	}
	return snd
}

func (a *Audio) loopingTrack(m *mixer.Mixer, p string, gain float32) *mixer.Track {
	audio := loadAudioFromFS(m, p)
	if audio == nil {
		return nil
	}
	t, err := m.CreateTrack()
	if err != nil {
		return nil
	}
	t.SetAudio(audio)
	t.SetLoops(-1)
	t.SetGain(gain)
	return t
}

// PlaySound plays one of a family of sound variants (name0 .. name(count-1)).
func (a *Audio) PlaySound(name string, count int) {
	if a.mixer == nil {
		return
	}
	variant := name + "0"
	if count > 1 {
		variant = name + strconv.Itoa(randIntn(count))
	}
	if snd, ok := a.sounds[variant]; ok && snd != nil {
		a.mixer.PlayAudio(snd)
	}
}

// PlayMusic switches to the named looping music track.
func (a *Audio) PlayMusic(name string) {
	a.StopMusic()
	if t, ok := a.music[name]; ok && t != nil {
		t.Play(0)
		a.currentMusic = t
	}
}

func (a *Audio) StopMusic() {
	if a.currentMusic != nil {
		a.currentMusic.Stop(0)
		a.currentMusic = nil
	}
}

// UpdateEngineSound selects the looping engine sample matching the current speed.
func (a *Audio) UpdateEngineSound(speed float64) {
	if a.engineTrack == nil || len(a.engineSounds) == 0 {
		return
	}
	idx := int(speed * 0.6)
	if idx > len(a.engineSounds)-1 {
		idx = len(a.engineSounds) - 1
	}
	if idx < 0 {
		idx = 0
	}
	if idx != a.currentEngineIdx {
		a.currentEngineIdx = idx
		if snd := a.engineSounds[idx]; snd != nil {
			a.engineTrack.SetAudio(snd)
			a.engineTrack.SetGain(0.3)
			a.engineTrack.SetLoops(-1)
			a.engineTrack.Play(100)
		}
	}
}

func (a *Audio) StopEngineSound() {
	if a.engineTrack != nil {
		a.engineTrack.Stop(0)
	}
	a.currentEngineIdx = -1
}

// SkidSound plays/fades the looping skid sound at the given volume (0 = off).
func (a *Audio) SkidSound(volume float64) {
	if a.skidTrack == nil {
		return
	}
	if volume > 0 {
		if !a.skidPlaying {
			a.skidTrack.SetLoops(-1)
			a.skidTrack.Play(100)
			a.skidPlaying = true
		}
		a.skidTrack.SetGain(float32(volume))
	} else if a.skidPlaying {
		a.skidTrack.Stop(250)
		a.skidPlaying = false
	}
}

func (a *Audio) Destroy() {
	if a.mixer != nil {
		a.mixer.Destroy()
	}
}
