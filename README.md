# legion

As in a **legion** of systems that are distributed golang system for data management, task processing and scalability. It uses Raft for distributed concensus and GRPC for communication.


## Parts of the system

### Dashboard
![alt text](https://i.imgur.com/yBc8Pit.png)
See Cluster Status, what nodes are online, who is the leader and what are the communication ports.

### Script Execution and workflows
Script creation, so you can write python code to create processing nodes in a workflow, then you can execute these in a distributed fashion accross all the systems.

#### Script Editor
![alt text](https://i.imgur.com/v0klRQX.png)

#### Workflow Creator
![alt text](https://i.imgur.com/4GIqs6N.png)

### SQL Shells for Local and Distributed Data
![alt text](https://i.imgur.com/FwQtX68.png)
