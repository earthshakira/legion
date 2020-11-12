// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"

// 	"github.com/graphql-go/graphql"
// 	"github.com/graphql-go/handler"
// )

// type Tutorial struct {
// 	ID       int64
// 	Title    string
// 	Author   Author
// 	Comments []Comment
// }

// type Author struct {
// 	Name      string
// 	Tutorials []int
// }

// type Comment struct {
// 	Body string
// }

// func populate() []Tutorial {
// 	author := &Author{Name: "Elliot Forbes", Tutorials: []int{1}}
// 	tutorial := Tutorial{
// 		ID:     1,
// 		Title:  "Go GraphQL Tutorial",
// 		Author: *author,
// 		Comments: []Comment{
// 			Comment{Body: "First Comment"},
// 		},
// 	}

// 	var tutorials []Tutorial
// 	tutorials = append(tutorials, tutorial)

// 	return tutorials
// }

// func main() {
// 	tutorials := populate()

// 	var commentType = graphql.NewObject(
// 		graphql.ObjectConfig{
// 			Name: "Comment",
// 			// we define the name and the fields of our
// 			// object. In this case, we have one solitary
// 			// field that is of type string
// 			Fields: graphql.Fields{
// 				"body": &graphql.Field{
// 					Type: graphql.String,
// 				},
// 			},
// 		},
// 	)

// 	var authorType = graphql.NewObject(
// 		graphql.ObjectConfig{
// 			Name: "Author",
// 			Fields: graphql.Fields{
// 				"Name": &graphql.Field{
// 					Type: graphql.String,
// 				},
// 				"Tutorials": &graphql.Field{
// 					// we'll use NewList to deal with an array
// 					// of int values
// 					Type: graphql.NewList(graphql.Int),
// 				},
// 			},
// 		},
// 	)

// 	var tutorialType = graphql.NewObject(
// 		graphql.ObjectConfig{
// 			Name: "Tutorial",
// 			Fields: graphql.Fields{
// 				"id": &graphql.Field{
// 					Type: graphql.Int,
// 				},
// 				"title": &graphql.Field{
// 					Type: graphql.String,
// 				},
// 				"author": &graphql.Field{
// 					// here, we specify type as authorType
// 					// which we've already defined.
// 					// This is how we handle nested objects
// 					Type: authorType,
// 				},
// 				"comments": &graphql.Field{
// 					Type: graphql.NewList(commentType),
// 				},
// 			},
// 		},
// 	)

// 	fields := graphql.Fields{
// 		"tutorial": &graphql.Field{
// 			Type: tutorialType,
// 			// it's good form to add a description
// 			// to each field.
// 			Description: "Get Tutorial By ID",
// 			// We can define arg	uments that allow us to
// 			// pick specific tutorials. In this case
// 			// we want to be able to specify the ID of the
// 			// tutorial we want to retrieve
// 			Args: graphql.FieldConfigArgument{
// 				"id": &graphql.ArgumentConfig{
// 					Type: graphql.Int,
// 				},
// 			},
// 			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
// 				// take in the ID argument
// 				fmt.Println(p)
// 				id, ok := p.Args["id"].(int)
// 				if ok {
// 					// Parse our tutorial array for the matching id
// 					for _, tutorial := range tutorials {
// 						if int(tutorial.ID) == id {
// 							// return our tutorial
// 							return tutorial, nil
// 						}
// 					}
// 				}
// 				return nil, nil
// 			},
// 		},
// 		// this is our `list` endpoint which will return all
// 		// tutorials available
// 		"list": &graphql.Field{
// 			Type:        graphql.NewList(tutorialType),
// 			Description: "Get Tutorial List",
// 			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
// 				return tutorials, nil
// 			},
// 		},
// 	}

// 	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
// 	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
// 	schema, err := graphql.NewSchema(schemaConfig)
// 	if err != nil {
// 		log.Fatalf("failed to create new schema, error: %v", err)
// 	}

// 	h := handler.New(&handler.Config{
// 		Schema:   &schema,
// 		Pretty:   true,
// 		GraphiQL: true,
// 	})

// 	http.Handle("/graphql", h)
// 	http.ListenAndServe(":8080", nil)
// }

package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/serf/serf"
)

var (
	members  string
	serfPort int
)

func init() {
	fmt.Println("Executed Init")
	flag.StringVar(&members, "members", "", "127.0.0.1:1111,127.0.0.1:2222")
	flag.IntVar(&serfPort, "serfPort", 0, "1111")
}

func main() {

	flag.Parse()

	var peers []string

	if members != "" {
		peers = strings.Split(members, ",")
	}

	ip := "192.168.0.155"

	// if err != nil {
	// 	log.Fatal(err)
	// }

	serfEvents := make(chan serf.Event, 16)

	memberlistConfig := memberlist.DefaultLANConfig()
	memberlistConfig.BindAddr = ip
	memberlistConfig.BindPort = serfPort
	memberlistConfig.LogOutput = os.Stdout
	serfConfig := serf.DefaultConfig()
	serfConfig.NodeName = fmt.Sprintf("%s:%d", ip, serfPort)
	serfConfig.EventCh = serfEvents
	serfConfig.MemberlistConfig = memberlistConfig
	serfConfig.LogOutput = os.Stdout

	s, err := serf.Create(serfConfig)

	if err != nil {
		log.Fatal(err)
	}

	// Join an existing cluster by specifying at least one known member.
	if len(peers) > 0 {

		_, err = s.Join(peers, false)

		if err != nil {
			log.Fatal(err)
		}
	}

	workDir, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	raftPort := serfPort + 1

	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%d", ip, raftPort))))

	dataDir := filepath.Join(workDir, id)

	err = os.RemoveAll(dataDir + "/")

	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(dataDir, 0777)

	if err != nil {
		log.Fatal(err)
	}

	raftDBPath := filepath.Join(dataDir, "raft.db")

	raftDB, err := raftboltdb.NewBoltStore(raftDBPath)

	if err != nil {
		log.Fatal(err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(dataDir, 1, os.Stdout)

	if err != nil {
		log.Fatal(err)
	}

	raftAddr := ip + ":" + strconv.Itoa(raftPort)

	trans, err := raft.NewTCPTransport(raftAddr, nil, 3, 10*time.Second, os.Stdout)

	if err != nil {
		log.Fatal(err)
	}

	c := raft.DefaultConfig()
	c.LogOutput = os.Stdout
	c.LocalID = raft.ServerID(raftAddr)

	r, err := raft.NewRaft(c, &fsm{}, raftDB, raftDB, snapshotStore, trans)

	if err != nil {
		log.Fatal(err)
	}

	bootstrapConfig := raft.Configuration{
		Servers: []raft.Server{
			{
				Suffrage: raft.Voter,
				ID:       raft.ServerID(raftAddr),
				Address:  raft.ServerAddress(raftAddr),
			},
		},
	}

	// Add known peers to bootstrap
	for _, node := range peers {

		if node == raftAddr {
			continue
		}

		bootstrapConfig.Servers = append(bootstrapConfig.Servers, raft.Server{
			Suffrage: raft.Voter,
			ID:       raft.ServerID(node),
			Address:  raft.ServerAddress(node),
		})
	}

	f := r.BootstrapCluster(bootstrapConfig)

	if err := f.Error(); err != nil {
		log.Fatalf("error bootstrapping: %s", err)
	}

	ticker := time.NewTicker(3 * time.Second)
	wassup := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-wassup.C:
			if serfPort == 1111 {
				resp, err := s.Query("Say Something", []byte("My Payload"), nil)
				if err != nil {
					log.Fatal(err)
				}
				ch := resp.ResponseCh()
				go func() {
					for c := range ch {
						fmt.Println("Receiving Channel Value: ", string(c.Payload))
					}
					resp.Close()
					fmt.Println("Go Func is done")
				}()
			}
		case <-ticker.C:
			future := r.VerifyLeader()

			fmt.Printf("Showing peers known by %s:\n", raftAddr)

			if err = future.Error(); err != nil {
				fmt.Println("Node is a follower")
			} else {
				fmt.Println("Node is leader")
			}

			cfuture := r.GetConfiguration()

			if err = cfuture.Error(); err != nil {
				log.Fatalf("error getting config: %s", err)
			}

			configuration := cfuture.Configuration()

			for _, server := range configuration.Servers {
				fmt.Println(server.Address)
			}

		case ev := <-serfEvents:

			leader := r.VerifyLeader()

			if memberEvent, ok := ev.(serf.MemberEvent); ok {

				for _, member := range memberEvent.Members {

					changedPeer := member.Addr.String() + ":" + strconv.Itoa(int(member.Port+1))

					if memberEvent.EventType() == serf.EventMemberJoin {

						if leader.Error() == nil {
							f := r.AddVoter(raft.ServerID(changedPeer), raft.ServerAddress(changedPeer), 0, 0)

							if f.Error() != nil {
								log.Fatalf("error adding voter: %s", err)
							}

						}

					} else if memberEvent.EventType() == serf.EventMemberLeave || memberEvent.EventType() == serf.EventMemberFailed || memberEvent.EventType() == serf.EventMemberReap {

						if leader.Error() == nil {

							f := r.RemoveServer(raft.ServerID(changedPeer), 0, 0)

							if f.Error() != nil {
								log.Fatalf("error removing server: %s", err)
							}
						}
					}
				}
			} else if ev.EventType() == serf.EventUser {
				fmt.Println("Got Event:", ev)
				fmt.Println()
			} else if ev.EventType() == serf.EventQuery {
				fmt.Println("Got Query:", ev)
				t, ok := ev.(*serf.Query)
				fmt.Println(t, ok)
				result := fmt.Sprintf("Hello from %d", raftPort)
				t.Respond([]byte(result))
				fmt.Println("Response Sent")
			}

		}
	}
}

type fsm struct {
}

func (f *fsm) Apply(*raft.Log) interface{} {
	return nil
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

func (f *fsm) Restore(io.ReadCloser) error {
	return nil
}
