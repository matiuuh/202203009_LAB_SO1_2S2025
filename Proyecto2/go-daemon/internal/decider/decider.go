package decider

import (
	"sort"
	"strconv"
	"strings"

	"proyecto2/daemon/internal/proc"
)

// candidato consolidado por contenedor
type cand struct {
	ID       string
	Name     string
	CPUPct   float64
	MemPct   float64
	RSS      uint64
	VSZ      uint64
	ShimPID  int
	ShimName string
}

type Decision struct {
	KeepIDs []string
	KillIDs []string
	Reason  map[string]string // reason[cid] = explicación corta
}

// PickKeepSet selecciona 2 "altos" + 3 "bajos" y devuelve qué matar (respetando protegidos).
// Reglas:
// - Protegidos nunca van en KillIDs (aunque no estén en Keep).
// - Si hay <5 contenedores, no matamos (Keep = todo, Kill = vacío).
// - "Altos": mayor CPU, luego Mem, luego RSS, luego VSZ.
// - "Bajos": menor CPU, luego Mem, luego RSS, luego VSZ.
func PickKeepSet(snap proc.ContSnapshot, protectIDs []string, protectNames []string) Decision {
	// normaliza listas de protección
	protID := make(map[string]struct{})
	for _, id := range protectIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			protID[id] = struct{}{}
		}
	}
	var protNames []string
	for _, n := range protectNames {
		n = strings.ToLower(strings.TrimSpace(n))
		if n != "" {
			protNames = append(protNames, n)
		}
	}

	// indexa por ContainerID (si viene vacío, cae a nombre/pid como fallback)
	byID := make(map[string]cand)
	for _, p := range snap.Processes {
		id := canonicalID(p)
		c := cand{
			ID:       id,
			Name:     p.Name,
			CPUPct:   p.CPUPct,
			MemPct:   p.MemPct,
			RSS:      p.RSS,
			VSZ:      p.VSZ,
			ShimPID:  p.ShimPID,
			ShimName: p.ShimName,
		}
		if prev, ok := byID[id]; ok {
			if better(c, prev) {
				byID[id] = c
			}
		} else {
			byID[id] = c
		}
	}

	// arma el slice de candidatos
	cands := make([]cand, 0, len(byID))
	for _, c := range byID {
		cands = append(cands, c)
	}

	total := len(cands)
	dec := Decision{Reason: make(map[string]string)}

	// si no hay suficientes, mantenemos todo
	if total <= 5 {
		for _, c := range cands {
			dec.KeepIDs = append(dec.KeepIDs, c.ID)
			dec.Reason[c.ID] = "insufficient_containers"
		}
		// KillIDs vacío
		return dec
	}

	// ordenamientos
	highs := make([]cand, len(cands))
	copy(highs, cands)
	sort.SliceStable(highs, func(i, j int) bool {
		// descendente: CPU, Mem, RSS, VSZ
		if highs[i].CPUPct != highs[j].CPUPct {
			return highs[i].CPUPct > highs[j].CPUPct
		}
		if highs[i].MemPct != highs[j].MemPct {
			return highs[i].MemPct > highs[j].MemPct
		}
		if highs[i].RSS != highs[j].RSS {
			return highs[i].RSS > highs[j].RSS
		}
		return highs[i].VSZ > highs[j].VSZ
	})

	lows := make([]cand, len(cands))
	copy(lows, cands)
	sort.SliceStable(lows, func(i, j int) bool {
		// ascendente: CPU, Mem, RSS, VSZ
		if lows[i].CPUPct != lows[j].CPUPct {
			return lows[i].CPUPct < lows[j].CPUPct
		}
		if lows[i].MemPct != lows[j].MemPct {
			return lows[i].MemPct < lows[j].MemPct
		}
		if lows[i].RSS != lows[j].RSS {
			return lows[i].RSS < lows[j].RSS
		}
		return lows[i].VSZ < lows[j].VSZ
	})

	keepSet := make(map[string]struct{})

	// helper: protegido por ID o por substring del nombre
	isProtected := func(c cand) bool {
		if _, ok := protID[c.ID]; ok {
			return true
		}
		name := strings.ToLower(c.Name)
		for _, sub := range protNames {
			if sub != "" && strings.Contains(name, sub) {
				return true
			}
		}
		return false
	}

	// elige 2 altos
	for _, c := range highs {
		if len(keepSet) >= 2 {
			break
		}
		keepSet[c.ID] = struct{}{}
		dec.Reason[c.ID] = "high_rank"
	}

	// elige 3 bajos (sin duplicar)
	for _, c := range lows {
		if len(keepSet) >= 5 {
			break
		}
		if _, ok := keepSet[c.ID]; ok {
			continue
		}
		keepSet[c.ID] = struct{}{}
		if _, ok := dec.Reason[c.ID]; !ok {
			dec.Reason[c.ID] = "low_rank"
		}
	}

	// pasa keepSet a slice
	for id := range keepSet {
		dec.KeepIDs = append(dec.KeepIDs, id)
	}

	// calcula KillIDs = todos los demás (pero NUNCA protegidos)
	for _, c := range cands {
		if _, ok := keepSet[c.ID]; ok {
			continue
		}
		if isProtected(c) {
			dec.Reason[c.ID] = "protected"
			continue
		}
		dec.KillIDs = append(dec.KillIDs, c.ID)
		dec.Reason[c.ID] = "not_in_keep"
	}

	// ordena para tener salida determinista
	sort.Strings(dec.KeepIDs)
	sort.Strings(dec.KillIDs)

	return dec
}

func better(a, b cand) bool {
	// criterio "mejor" para representar al contenedor
	if a.CPUPct != b.CPUPct {
		return a.CPUPct > b.CPUPct
	}
	if a.MemPct != b.MemPct {
		return a.MemPct > b.MemPct
	}
	if a.RSS != b.RSS {
		return a.RSS > b.RSS
	}
	return a.VSZ > b.VSZ
}

func canonicalID(p proc.ContProc) string {
	// Preferimos ContainerID; si viene vacío, caemos a shim o nombre con PID
	if p.ContainerID != "" {
		return p.ContainerID
	}
	if p.ShimPID != 0 {
		return "shim:" + itoa(p.ShimPID)
	}
	if p.PID != 0 {
		return "pid:" + itoa(p.PID)
	}
	if p.Name != "" {
		return "name:" + strings.ToLower(p.Name)
	}
	return "unknown"
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
