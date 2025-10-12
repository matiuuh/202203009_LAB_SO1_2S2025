package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"proyecto2/daemon/internal/db"
	"proyecto2/daemon/internal/decider"
	"proyecto2/daemon/internal/proc"
)

const (
	logPath           = "/var/log/daemon_202203009.log"
	loopEvery         = 20 * time.Second
	loadModulesScript = "/home/matius/Descargas/SistemasOperativos/Proyecto1/Proyecto1/202203009_LAB_SO1_2S2025/Proyecto2/bash/load_modules.sh"
	DBPath            = "/var/lib/proyecto2/metrics.db"
	installCronScript = "/home/matius/Descargas/SistemasOperativos/Proyecto1/Proyecto1/202203009_LAB_SO1_2S2025/Proyecto2/bash/install_cron.sh"
	removeCronScript  = "/home/matius/Descargas/SistemasOperativos/Proyecto1/Proyecto1/202203009_LAB_SO1_2S2025/Proyecto2/bash/remove_cron.sh"
)

var logger *log.Logger

// ---- Config de decisiones (apagadas por defecto) ----
var (
	deciderEnabled  = envBool("DECIDER_ENABLED", true)   // activar = true/1/yes
	protectNamesCSV = getenv("PROTECT_NAMES", "grafana") // substrings separados por coma
	protectIDsCSV   = getenv("PROTECT_IDS", "")          // IDs exactos separados por coma
	enforceEveryN   = envInt("ENFORCE_EVERY_N", 1)       // aplicar cada N vueltas
	enforceTick     = 0                                  // contador de vueltas
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func envBool(k string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	if v == "1" || v == "true" || v == "yes" {
		return true
	}
	if v == "0" || v == "false" || v == "no" {
		return false
	}
	return def
}
func envInt(k string, def int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return def
}
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func initLogging() {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("no pude abrir %s: %v (logeando a stderr)", logPath, err)
		logger = log.New(os.Stderr, "[daemon] ", log.LstdFlags)
		return
	}
	logger = log.New(f, "[daemon] ", log.LstdFlags)
}

func runScript(path string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ensureDir(dir string) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Fatalf("mkdir %s: %v", dir, err)
	}
}

func main() {
	initLogging()
	logger.Println("Daemon iniciado (Iteración 1/2: módulos, /proc, DB; decisiones apagadas)")

	/*
		Usamos 10*time.second porque con eso le decimos al programa que ese es el tiempo
		maximo que puede tardar en ejecutar el script, si tarda mas de ese tiempo
		entonces se mata el proceso y se devuelve un error.
	*/

	// 1) Cargar módulos del kernel vía script
	if err := runScript(loadModulesScript, 60*time.Second); err != nil {
		logger.Printf("load_modules.sh falló: %v", err)
	}

	// Instalar cron para spawn de contenedores
	if err := runScript(installCronScript, 10*time.Second); err != nil {
		logger.Printf("install_cron.sh falló: %v", err)
	}
	// asegurar que se quite al salir del daemon
	/*
		este defer se asegura de que al finalizar el programa el cron sea removido,
		de esta manera garantizamos que no queden procesos de cron corriendo
		despues de que el daemon haya finalizado
	*/
	defer func() {
		if err := runScript(removeCronScript, 10*time.Second); err != nil {
			logger.Printf("remove_cron.sh falló: %v", err)
		}
	}()

	// 2) Validar /proc
	checkPath(proc.SysProcPath)
	checkPath(proc.ContProcPath)

	// 3) Abrir DB
	ensureDir("/var/lib/proyecto2")
	store, err := db.Open(DBPath)
	if err != nil {
		logger.Fatalf("db open: %v", err)
	}
	defer store.Close()

	// 4) Señales + loop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(loopEvery)
	defer ticker.Stop()

	// Primera lectura inmediata
	runOnce(store)

	for {
		select {
		case <-ticker.C:
			runOnce(store)
		case s := <-stop:
			logger.Printf("Recibida señal %s, saliendo...", s)
			return
		}
	}
}

func checkPath(p string) {
	if _, err := os.Stat(p); err != nil {
		logger.Printf("ADVERTENCIA: no existe %s (%v). ¿Cargaste el módulo del kernel?", p, err)
	}
}

func runOnce(store *db.Store) {
	// 1) timestamp fijo para ESTA vuelta
	ts := time.Now().Unix()

	// 2) transacción para minimizar locks y acelerar
	tx, err := store.Begin()
	if err != nil {
		logger.Printf("db begin: %v", err)
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// ---- SYS ----
	var sysSnap proc.SysSnapshot
	if err := proc.ReadJSON(proc.SysProcPath, &sysSnap); err != nil {
		logger.Printf("error leyendo %s: %v", proc.SysProcPath, err)
	} else {
		logger.Printf("SYS: Totalram=%d KB Freeram=%d KB Procs=%d ProcList=%d",
			sysSnap.Totalram, sysSnap.Freeram, sysSnap.Procs, len(sysSnap.Processes))

		//para la ram del sistema
		if stmtHost, err := tx.Prepare(`INSERT INTO host_metrics (ts,totalram_kb,freeram_kb) VALUES (?,?,?)`); err == nil {
			_, _ = stmtHost.Exec(ts, sysSnap.Totalram, sysSnap.Freeram)
			_ = stmtHost.Close()
		} else {
			logger.Printf("prep host_metrics: %v", err)
		}
		stmtSys, _ := tx.Prepare(`INSERT INTO system_metrics
            (ts,pid,name,state,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
            VALUES (?,?,?,?,?,?,?,?,?)`)
		if stmtSys != nil {
			for _, p := range sysSnap.Processes {
				if _, err := stmtSys.Exec(ts, p.PID, p.Name, p.State, p.Cmdline, p.VSZ, p.RSS, p.MemPct, p.CPUPct); err != nil {
					logger.Printf("db sys insert pid=%d err=%v", p.PID, err)
				}
			}
			_ = stmtSys.Close()
		}
	}

	// ---- CONT ----
	var contSnap proc.ContSnapshot
	if err := proc.ReadJSON(proc.ContProcPath, &contSnap); err != nil {
		logger.Printf("error leyendo %s: %v", proc.ContProcPath, err)
	} else {
		logger.Printf("CONTAINERS: Totalram=%d KB Freeram=%d KB Entries=%d",
			contSnap.Totalram, contSnap.Freeram, len(contSnap.Processes))

		if len(contSnap.Processes) > 0 {
			top := append([]proc.ContProc{}, contSnap.Processes...)
			sort.Slice(top, func(i, j int) bool { return top[i].CPUPct > top[j].CPUPct })
			for i := 0; i < len(top) && i < 3; i++ {
				logger.Printf("TOPCPU[%d] cid=%s name=%s cpu=%.2f mem=%.1f",
					i+1, shortCID(top[i].ContainerID), top[i].Name, top[i].CPUPct, top[i].MemPct)
			}
		}

		stmtCont, _ := tx.Prepare(`INSERT INTO container_metrics
            (ts,container_id,shim_pid,shim_name,pid,name,cmdline,vsz_kb,rss_kb,mem_pct,cpu_pct)
            VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
		if stmtCont != nil {
			for _, c := range contSnap.Processes {
				if _, err := stmtCont.Exec(ts, c.ContainerID, c.ShimPID, c.ShimName, c.PID, c.Name, c.Cmdline, c.VSZ, c.RSS, c.MemPct, c.CPUPct); err != nil {
					logger.Printf("db cont insert cid=%s err=%v", shortCID(c.ContainerID), err)
				}
			}
			_ = stmtCont.Close()
		}

		// ---- DECISIÓN (solo calcular aquí; ejecutar kills después del commit) ----
		var decision *decider.Decision
		if deciderEnabled && len(contSnap.Processes) > 0 {
			enforceTick++
			if enforceTick%enforceEveryN == 0 {
				protNames := splitCSV(protectNamesCSV)
				protIDs := splitCSV(protectIDsCSV)
				dec := decider.PickKeepSet(contSnap, protIDs, protNames)
				decision = &dec
				logger.Printf("DECIDER: keep=%d kill=%d (tick=%d/%d)",
					len(dec.KeepIDs), len(dec.KillIDs), enforceTick, enforceEveryN)
			}
		}

		// 3) fin de la vuelta: commit (cerramos rápido la transacción)
		if err := tx.Commit(); err != nil {
			logger.Printf("db commit: %v", err)
			return
		}
		committed = true

		// 4) Ejecutar kills *después* del commit para no bloquear la DB
		if decision != nil && len(decision.KillIDs) > 0 {
			killContainers(store, decision.KillIDs, decision.Reason)
		}
		return
	}

	// si no hubo contSnap válido, igual cerramos transacción
	if err := tx.Commit(); err != nil {
		logger.Printf("db commit: %v", err)
		return
	}
	committed = true
}

func killContainers(store *db.Store, ids []string, reason map[string]string) {
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", id)
		out, err := cmd.CombinedOutput()
		cancel()

		details := strings.TrimSpace(string(out))
		rsn := reason[id]
		if err != nil {
			logger.Printf("docker rm -f %s FAILED: %v | %s", id, err, details)
			_ = store.LogAction("kill_failed", id, rsn, details)
		} else {
			logger.Printf("docker rm -f %s OK | %s", id, details)
			_ = store.LogAction("kill", id, rsn, details)
		}
	}
}

func shortCID(id string) string {
	if len(id) > 12 {
		return fmt.Sprintf("%s…", id[:12])
	}
	return id
}
