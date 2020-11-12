package execution

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/sophron-dev-works/legion/ui"
)

func trialRun(file string, wg *sync.WaitGroup) (string, string, string) {
	wg.Add(1)
	goExecutable, _ := exec.LookPath("python3")
	var sout, serr bytes.Buffer
	cmdGoVer := &exec.Cmd{
		Path:   goExecutable,
		Args:   []string{goExecutable, file},
		Stdout: &sout,
		Stderr: &serr,
	}
	err := cmdGoVer.Run()
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	fmt.Println(serr.String())
	wg.Done()
	return cmdGoVer.String(), string(sout.String()), string(serr.String())
}

type ExecutionEngine struct {
	db *badger.DB
	vs ui.ViewState
}

func (ee *ExecutionEngine) defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"vs": ee.vs,
	}
}

func (ee *ExecutionEngine) Create(iw router.Party, vs ui.ViewState) {
	ee.vs = vs

	ee.db, _ = badger.Open(badger.DefaultOptions("/tmp/scriptStore"))
	iw.Get("/", func(ctx iris.Context) {
		ctx.View("execution.html", ee.defaultConfig())
	})

	iw.Get("/view/{name}", func(ctx iris.Context) {
		// TODO: handle errors
		scriptPayload, _ := ee.getScript(ctx.Params().Get("name"))
		data := ee.defaultConfig()
		fmt.Println(scriptPayload)
		data["script"] = scriptPayload
		ctx.View("execution.html", data)
	})
	iw.Get("/list", ee.listAllScripts)
	iw.Post("/execute", ee.executeScript)
	iw.Post("/save", ee.saveScript)
}

func (ee *ExecutionEngine) getScript(name string) (ScriptPayload, error) {
	var sp ScriptPayload
	err := ee.db.View(func(txn *badger.Txn) error {
		// TODO: handle Errors
		item, _ := txn.Get([]byte("script." + name))
		_ = item.Value(func(val []byte) error {
			sp.FromBytes(val)
			return nil
		})
		return nil
	})
	if err != nil {
		return sp, err
	}
	return sp, nil
}

func (ee *ExecutionEngine) listAllScripts(ctx iris.Context) {
	var scripts []ScriptPayload
	ee.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("script.")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				var sp ScriptPayload
				sp.FromBytes(v)
				sp.Text = ""
				scripts = append(scripts, sp)
				fmt.Println(string(k), sp)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	ctx.JSON(scripts)
}
func (ee *ExecutionEngine) saveScript(ctx iris.Context) {
	var b ScriptPayload
	err := ctx.ReadJSON(&b)
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to parse payload").DetailErr(err))
		return
	}

	b.Modified = time.Now()
	err = ee.db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("script."+b.Name), b.Bytes())
		return err
	})
	ctx.StatusCode(iris.StatusCreated)

}

func (ee *ExecutionEngine) executeScript(ctx iris.Context) {
	var b ScriptPayload
	err := ctx.ReadJSON(&b)
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to execute script").DetailErr(err))
		return
	}

	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to parse payload").DetailErr(err))
		return
	}

	dir, err := ioutil.TempDir("/tmp", "script")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	fmt.Println(dir)
	fileName := dir + "/script.py"
	fmt.Println(b.Text)
	err = ioutil.WriteFile(fileName, []byte(b.Text), 0644)
	out, err := exec.Command("python3", fileName).Output()
	fmt.Println(out)
	var wg sync.WaitGroup
	cmd, op, er := trialRun(fileName, &wg)
	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(ScriptOutput{cmd, er, op})
}
