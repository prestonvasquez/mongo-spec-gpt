Prompt: "Generate a mermaid diagram for rtt monitor"

mongo-spec-gpt:

```mermaid
flowchart TD
  A[Start RTT Monitor Thread]
  B[Acquire Dedicated Connection]
  C[Wait for next interval]
  D[Record T1 and send “hello” command]
  E[Receive “hello” response and record T2]
  F[Compute RTT = T2 – T1]
  G[Update Server Description with new RTT]
  H{Continue?}
  A --> B --> C
  C --> D --> E --> F --> G
  G --> H
  H -- Yes --> C
  H -- No --> I[Stop RTT Monitor Thread]
```

MongoGPT:

```mermaid
graph TD  
    A[RTT Monitor Start] -->|Initiate Request| B[Send Request Packet]  
    B -->|Record Timestamp T1| C[Network Transmission]  
    C -->|Reach Destination| D[Server or Endpoint]  
    D -->|Process Request| E[Generate Response Packet]  
    E -->|Send Response Back| F[Network Transmission Return]  
    F -->|Record Timestamp T2| G[RTT Calculation]  
    G -->|RTT = T2 - T1| H[Log RTT Value]  
    H -->|Monitor RTT Consistently| I[Alert if RTT Exceeds Threshold]  
```
Prompt: "Generate a mermaid diagram for multi-threaded server selection"

### mongo-spec-gpt

Here’s a Mermaid flowchart illustrating a multi‐threaded (or asynchronous) server-selection loop. It uses a single global lock and a condition variable for waiting threads:

```mermaid
flowchart TD
    A[Start serverSelection] --> B[Acquire global lock]
    B --> C{Run ServerSelectionIteration}
    C -->|Found suitable servers| D[Filter by latency window]
    D --> E[Pick one server at random]
    E --> F[Log “Server selection succeeded”]
    F --> G[Release global lock]
    G --> H[Return selected server]

    C -->|No suitable server| I[Mark topology as stale]
    I --> J[Release global lock]
    J --> K[Wait on condition variable or timeout]
    K --> B

    style A fill:#f9f,stroke:#333,stroke-width:2px
    style H fill:#9f9,stroke:#333,stroke-width:2px
    style K fill:#ff9,stroke:#333,stroke-width:2px
```

Explanation of the steps:

 1. **Acquire global lock**: Ensure only one thread at a time performs the iteration.
 2. **ServerSelectionIteration()**: Check the current topology and your read/write preferences.
 3. **If servers found**
    • Filter them by the configured latency window.
    • Pick one at random, log success, release the lock, and return it.
 4. **If none found**
    • Mark the local topology description as stale.
    • Release the lock and wait on a condition variable (or timeout) until another thread or background monitor updates the topology.
    • Once woken, re-acquire the lock and retry the loop.
  
Prompt: "Generate a mermaid sequence diagram for multi-threaded server selection"

### mongo-spec-gpt
```mermaid
sequenceDiagram
    participant Client
    participant Selector
    participant Topology
    participant Timer
    participant Logger
    participant Server

    Client->>Selector: selectServer()
    loop retry until success or timeout
        Note over Selector: Step 2–7: filter and sort candidates
        alt candidate found (Step 8)
            Selector->>Server: incrementOperationCount()
            Server-->>Selector: selected
            Selector->>Logger: log("Server selection succeeded")
            Selector-->>Client: return Server
        else no candidate
            Note over Selector: Step 9: request immediate topology check
            Selector->>Topology: requestImmediateCheck()
            Note right of Selector: block until <br/> topologyChanged OR timeout
            par wait for events
                Topology-->>Selector: topologyChanged
            and
                Timer-->>Selector: timeoutElapsed
            end
            alt timed out (Step 10)
                Selector->>Logger: log("Server selection failed")
                Selector-->>Client: raise ServerSelectionError
            else topology changed (Step 11)
                Note right of Selector: goto Step 2 (retry)
            end
        end
    end
```

Notes:

- The outer `loop` represents retrying selection until either a server is found or the operation times out.
- The first branch (`alt candidate found`) corresponds to Step 8: incrementing the server’s `operationCount`, logging success, and returning the server.
- The second branch (`else no candidate`) corresponds to Steps 9–11:
  - Requesting an immediate topology check.
  - Blocking (waiting in parallel) for either a topology change or the timeout.
  - On timeout, logging failure and raising an error (Step 10).
  - On topology change, retrying the selection (Step 11).


```mermaid
flowchart TD
  %% Multi-threaded Server Selection Algorithm
  Start["Start: Record server selection start time;\nlog ‘Server selection started’"]
    --> Step2{"Topology wire version valid?"}

  Step2 -- No --> FailWire["Log ‘Server selection failed’;\nraise wire version error"]
  FailWire --> EndFail1["End"]

  Step2 -- Yes --> Step3["Find suitable servers by topology & operation type;\nhandle deprioritized servers"]
  Step3 --> Step4["Filter suitable servers with\napplication-provided selector"]
  Step4 --> Step5{"Any suitable servers?"}

  Step5 -- No --> Waiting["Log ‘Waiting for suitable server to become available’\n(if not already logged)"]
  Waiting --> TopoCheck["Request immediate topology check;\nblock until topology changes or timeout"]
  TopoCheck --> TimedOut{"Server selection timed out?"}
  TimedOut -- Yes --> FailTimeout["Log ‘Server selection failed’;\nraise server selection error"]
  FailTimeout --> Step2
  TimedOut -- No --> Step2

  Step5 -- Yes --> Step6["Filter suitable servers by\nlatency window"]
  Step6 --> Step7{">1 server in latency window?"}
  Step7 -- Yes --> RandPick["Randomly pick two servers"]
  Step7 -- No --> SinglePick["Select the single server in window"]
  RandPick --> CompareOp{"Compare two servers by operationCount\n(lowest wins; tie ⇒ arbitrary)"}
  CompareOp --> SelectServer
  SinglePick --> SelectServer

  SelectServer["Increment operationCount;\nlog ‘Server selection succeeded’;\nreturn selected server"]
    --> EndSuccess["End"]
```

Generate a mermaid diagram for server selection with the following context: 

MongoDB’s drivers use a standardized “server selection” algorithm whenever they need to pick a server for an operation (read or write).  In broad strokes the process is:

1. Maintain a Topology Description
   • Each driver continually heartbeats every known host in the seed list or discovered via replica‐set gossip or sharding metadata.
   • From those heartbeats it builds and keeps up-to-date a view of the topology: which servers are primaries, secondaries, routers, unknown, etc., and their round‐trip latencies.

2. Determine the Operation’s Requirements
   • Writes always require the primary.
   • Reads use the configured Read Preference (primary, primaryPreferred, secondary, secondaryPreferred, nearest).
   • The Read Preference may include tag sets (e.g. `{ dc: "east", rack: "1" }`) to constrain eligible servers.

3. Filter by Server Type
   • From the current topology view, throw away any servers whose type cannot satisfy the request (e.g. drop secondaries for a primary‐only read).

4. Filter by Tags (if any)
   • If the Read Preference has tag sets, eliminate any servers that don’t match at least one set.

5. Apply Latency Window (“nearest” / localThresholdMillis)
   • Of the remaining candidates, find the minimum observed heartbeat RTT.
   • Discard any server whose RTT is more than localThresholdMillis (default 15 ms) above that minimum.
   • This ensures the driver only picks “nearby” nodes, preventing it from sending reads to a very slow replica even if it technically matches tags and type.

6. Select One at Random
   • From the final filtered list, pick a server at random (uniformly).
   • Use that connection for the operation.

7. Timeout and Retry
   • If the list ever becomes empty, the driver will retry heartbeats and re-run selection until the serverSelectionTimeout (default 30 s) expires, at which point it raises a server selection timeout error.

By separating discovery (heartbeat monitoring) from selection (filter → latency window → random choice), MongoDB drivers can react quickly to topology changes while still consistently steering operations to healthy, low-latency servers that meet your read/write and tagging requirements.

Result:
```mermaid
flowchart TD
  %% Discovery
  A[Heartbeat Monitoring] --> B[Topology Description]

  %% Selection entry
  B --> C{Operation Type?}

  %% Write path
  C -->|Write| D[Filter: Primary Only]
  D --> E[Select Primary Server]
  E --> F[Use Connection]

  %% Read path
  C -->|Read| G[Filter: By Read Preference]
  G --> H{Has Tag Sets?}
  H -->|Yes| I[Filter by Tags]
  H -->|No| J[Skip Tag Filtering]
  I --> K[Filter by Latency Window]
  J --> K
  K --> L{Any Candidates?}
  L -->|Yes| M[Select Random Server]
  M --> F

  %% Retry / Timeout
  L -->|No| N[Retry heartbeat & re-select]
  N --> A
  N -->|On Timeout| O[ServerSelectionTimeout Error]
```

Explanation of the steps:

1. **Heartbeat Monitoring → Topology Description**
   Drivers continually heartbeat all known nodes and build a current view of the topology.

2. **Operation Type?**
   Branch to either a write or read selection path.

3. **Write Path**
   Always filter to the primary, then select and use it immediately.

4. **Read Path**
   a. Apply the configured read preference (primary, secondary, nearest, etc.).
   b. If there are tag sets, filter servers to those matching any tag set.
   c. Apply the latency window (localThresholdMillis) to drop high‐latency nodes.
   d. If candidates remain, pick one at random and use it.

5. **Retry / Timeout**
   If at any point the candidate list is empty, retry the selection loop (respecting minHeartbeatFrequencyMS) until serverSelectionTimeoutMS is reached, then raise a timeout error.

### Prompt (With Chaining): "How does server selection work? | Generate a mermaid diagram for server selection. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TB
    A[Record start time and log Server selection started] --> B{Is wire version valid}
    B -- No --> C[Error out and log Server selection failed]
    B -- Yes --> D[Build initial list of suitable servers based on topology type operation type and read or write preferences]
    D --> E{Is topology Sharded and deprioritized servers provided and no other suitable servers}
    E -- Yes --> F[Include deprioritized servers]
    E -- No --> G[Ignore deprioritized servers]
    F --> H[List of suitable servers]
    G --> H
    H --> I[Apply optional application provided selector]
    I --> J[Compute fastest round trip time add slack discard servers outside latency window]
    J --> K{Are there servers in latency window}
    K -- No --> L[Log Waiting for suitable server block or retry]
    L --> J
    K -- Yes --> M{Is there exactly one server}
    M -- Yes --> N[Select sole server]
    M -- No --> O[Pick two servers at random compare operationCount choose server with fewer operations]
    N --> P[Return selected server]
    O --> P
```

### Prompt (With Chaining): "How does rtt monitor  work? | Generate a mermaid diagram for rtt monitor. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart LR
    Start --> OpenConnectionPerServer
    OpenConnectionPerServer --> InitialHandshake
    InitialHandshake --> RecordFirstRTTSample
    RecordFirstRTTSample --> HeartbeatLoop

    subgraph HeartbeatLoop
      HLStart --> WaitHeartbeatInterval
      WaitHeartbeatInterval --> SendHelloCommand
      SendHelloCommand --> ReceiveHelloResponse
      ReceiveHelloResponse --> MeasureElapsedTime
      MeasureElapsedTime --> UpdateMovingAverageRTT
      UpdateMovingAverageRTT --> HLStart
    end

    UpdateMovingAverageRTT --> FeedServerSelection
```
----
```mermaid
flowchart LR
  subgraph RTT Monitor
    Start[Open dedicated connection per server]
    Tick[Heartbeat tick]
    Before[Timestamp before send]
    Send[Issue hello command]
    After[Timestamp after receive]
    Compute[Compute RTT difference]
    Store[Update server description with RTT]
  end

  Start --> Tick
  Tick --> Before
  Before --> Send
  Send --> After
  After --> Compute
  Compute --> Store
  Store --> Tick
  Store --> Selection

  Selection[Use RTT in server selection and local threshold]
```
----
```mermaid
flowchart TD
  A[Start RTT Monitor] --> B[Open Dedicated Connection to Server]
  B --> C[Compute Initial RTT from Handshake Hello]
  C --> D[Wait heartbeatFrequencyMS]
  D --> E[Send Hello on Dedicated Connection]
  E --> F[Record Send Timestamp]
  E --> G[Receive Hello Reply]
  G --> H[Record Receive Timestamp]
  F --> I[Compute Round Trip Time]
  H --> I
  I --> J[Update Running Average RTT]
  J --> D
```
----
```mermaid
flowchart LR
  A[Dedicated connection to each server]
  B[Periodic hello probes every heartbeatFrequencyMS]
  C[Timestamp send and receive]
  D[Compute new RTT sample]
  E[Update running average for server selection]
  F[Initial handshake provides initial sample]
  G[Ignore streaming hello responses for RTT]
  A --> B
  B --> C
  C --> D
  D --> E
  F --> E
  B --> G
```
----
```mermaid
flowchart TD
  A[Discover new server] --> B[Open dedicated connection]
  B --> C[Assign to RTT monitor thread]
  C --> D[Initial hello handshake]
  D --> E[Measure handshake duration]
  E --> F[Set initial RTT estimate]
  F --> G[Start periodic ping loop]
  G --> H[Sleep for heartbeatFrequencyMS]
  H --> I[Send hello on dedicated connection]
  I --> J[Receive reply]
  J --> K[Compute delta time]
  K --> L[Update running average RTT]
  L --> H
```
----
```mermaid
flowchart TD
    Start[Start]
    A[Dedicated connection per server]
    B[Periodic hello or legacyHello]
    C[Measure and record RTT]
    D[Update server description]
    E[Never reuse pool connections]

    Start --> A
    A --> B
    B --> C
    C --> D
    D --> B
    E -.-> A
    E -.-> B
    E -.-> C
```
----
```mermaid
flowchart TD
    A[Open dedicated connection for RTT checks]
    A --> B[Measure initial RTT from handshake]
    B --> C[Every heartbeatFrequencyMS issue hello or legacy hello]
    C --> D[Update moving average RTT]
    D --> E[Feed RTT into server selection logic]
    C -.-> F[Runs in background to isolate from application pool]
    F --> G[Application connection pool remains unaffected]
```
----
```mermaid
flowchart TD
  A[Startup] --> B[Open dedicated RTT connection per server]
  B --> C[Initial handshake measures RTT]
  C --> D[Record initial RTT]
  D --> E[Wait heartbeat interval]
  E --> F[Send hello or legacy hello on RTT connection]
  F --> G[Measure round trip time]
  G --> H[Update average RTT value]
  H --> I[Server selection uses RTT and local threshold]
  I --> E
```
----
```mermaid
flowchart TD
  A[Establish main connection] --> B[Open dedicated RTT connection]
  B --> C[Initial hello handshake]
  C --> D[Record initial RTT]
  D --> E[Start periodic timer]
  E --> F[Send hello command]
  F --> G[Receive response measure RTT]
  G --> H[Update rolling average RTT]
  H --> I[Server selection uses RTT plus threshold]
```

### Prompt (With Chaining): "How is SDAM supposed to function? | Generate a mermaid diagram for SDAM. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
  Start[Start] --> MonitorPerEndpoint[Per server monitor]
  MonitorPerEndpoint --> SendHelloPing[Send hello and ping commands]
  SendHelloPing --> UpdateServerDescription[Update ServerDescription]
  UpdateServerDescription --> TopologyAggregation[Aggregate Topology]
  TopologyAggregation --> LogMessage[Log standardized message]
  TopologyAggregation --> FireEvents[Fire SDAM events]
  LogMessage --> Application[Application consumes events]
  FireEvents --> Application
```
----

### Prompt (With Chaining): "How is rtt monitoring implemented? | Generate a mermaid diagram for rtt monitoring. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
graph TD
RTTThread[RTT monitoring thread] --> Conn[Dedicated RTT connection]
Conn --> SendHello[Send hello or legacy hello]
SendHello --> Recv[Receive hello response]
Recv --> Measure[Measure elapsed time]
Measure --> Update[Update server average RTT]
Update --> Wait[Wait heartbeatFrequencyMS]
Wait --> SendHello
```
----
### Prompt (With Chaining): "How is rtt monitoring implemented? | How should I implement rtt monitoring in an algorithm? | Generate a mermaid diagram for rtt monitoring. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
A[Open dedicated connection to server]
B[Perform initial hello handshake]
C[Measure initial RTT from handshake]
D[Set server RTT to initial RTT]
E[Enter periodic RTT monitoring loop]
F[Sleep for heartbeatFrequencyMS]
G[Send hello command]
H[Receive reply]
I[Compute elapsed RTT]
J[Update server RTT with EWMA]
K[Shutdown and close dedicated connection]
A --> B --> C --> D --> E
E --> F --> G --> H --> I --> J --> E
E --> K
```
----
### Prompt (With Chaining): "How should I implement rtt monitoring in an algorithm? | Generate a mermaid diagram for rtt monitoring. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
    A[Create RTT monitor for server]
    B[Open dedicated connection]
    C[Perform initial hello handshake]
    D[Measure initial RTT]
    E[Sleep for heartbeatFrequencyMS]
    F[Send hello command]
    G[Receive hello response]
    H[Update average RTT]
    I{Shutting down}
    J[Close dedicated connection]

    A --> B
    B --> C
    C --> D
    D --> E
    E --> F
    F --> G
    G --> H
    H --> I
    I -- Yes --> J
    I -- No --> E
```
----
### Prompt (With Chaining): "How should I implement server selection in an algorithm? | Generate a mermaid diagram for server selection. The content of your diagram should not contain any quotation marks or parentheses."
 ```mermaid
graph TD
  A[Record start time and log Server selection started] --> B{Topology wire version valid}
  B -- no --> C[Log Server selection failed and error]
  B -- yes --> D[Build initial suitable servers list]
  D --> E[Apply application selector callback]
  E --> F{Suitable servers remaining}
  F -- yes --> G[Compute minimum ping and apply latency window]
  G --> H{Client is multithreaded or single threaded}
  H -- multithreaded --> I[If only one server select it else pick two random servers compare operation counts select lower and increment count]
  H -- single threaded --> J[Select random server]
  I --> K[Log Server selection succeeded and return server]
  J --> K
  F -- no --> L[Log waiting for suitable server message once]
  L --> M[Trigger immediate topology refresh and wait]
  M --> N{Timeout expired}
  N -- yes --> O[Log Server selection failed and raise timeout error]
  N -- no --> B
```
----
```mermaid
flowchart TD
  A[Start server selection] --> B{Topology wire version valid}
  B -- no --> C[Log server selection failed and throw error]
  B -- yes --> D[Build initial candidate set by topology type operation type tag sets max staleness]
  D --> E[Apply custom server selector callback]
  E --> F[Measure ping times compute minimum RTT filter by latency threshold]
  F --> G{Suitable candidates exist}
  G -- yes --> L{Multithreaded client model}
  G -- no --> H[Log waiting for suitable server]
  H --> I[Mark topology stale request immediate topology check]
  I --> J{Timeout elapsed}
  J -- yes --> K[Log server selection failed throw timeout error]
  J -- no --> B

  L -- yes --> M{Only one server in latency window}
  L -- no --> N[Select one server at random]
  M -- yes --> M1[Select single server increment operation count]
  M -- no --> M2[Select two servers compare operation count select lower increment operation count]

  M1 --> O[Log server selection succeeded return server]
  M2 --> O
  N --> O
```
----
### Prompt (With Chaining): "How should I implement server discovery in an algorithm? | Generate a mermaid diagram for server discovery. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
  A[Start] --> B[Input seed list and global timeout]
  B --> C[Initialize knownServers to seed list]
  C --> D[Record startTime as now]
  D --> E{now minus startTime less than timeout}
  E -->|yes| F[For each address not yet probed send hello]
  F --> G[Wait with short timeout for replies]
  G --> H[For each successful reply parse fields and add new hosts]
  H --> I{New hosts discovered in this iteration}
  I -->|yes| E
  I -->|no| J[Exit loop]
  E -->|no| K[Initial topology ready]
  J --> K
  K --> L[End]
```
----
### Prompt (With Chaining): "How should I implement transactions  in an algorithm? | Generate a mermaid diagram for transactions. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TB
    A[Start] --> B[Start Transaction]
    B --> C[Run Transaction Logic]
    C --> D{Error Occurred}
    D -->|Transient Transaction Error| E[Abort Transaction]
    E --> B
    D -->|Other Error| F[Abort Transaction]
    F --> G[End with Error]
    D -->|No Error| H[Commit Transaction]
    H --> I{Commit Error Occurred}
    I -->|Unknown Transaction Commit Result| H
    I -->|Other Commit Error| J[End with Error]
    I -->|No Error| K[Success End]
```
----
### Prompt (With Chaining): "How should I implement the server handshake in an algorithm? | Generate a mermaid diagram for server handshakes. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
  A[Open Socket]
  B[Send Handshake with Operation Timeout]
  C{Handshake Error}
  D[Handle Handshake Failure]
  E[Parse Server Type]
  F{Versioned API and Legacy Hello Received}
  G{Server Type is Arbiter}
  H[Handshake Success]
  I[Perform Authentication Conversation]
  J{Authentication Error}
  K[Handle Authentication Failure]

  A --> B
  B --> C
  C -->|Yes| D
  C -->|No| E
  E --> F
  F -->|Yes| D
  F -->|No| G
  G -->|Yes| H
  G -->|No| I
  I --> J
  J -->|Yes| K
  J -->|No| H
```
----
### Prompt (With Chaining): "How should I implement CSOT in an algorithm? | Generate a mermaid diagram for CSOT. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
  Start[Start operation] --> Init[Record start time and deadline]
  Init --> Op[Execute operation]
  Op --> Error{Did operation error}
  Error -- No --> Success[Return result]
  Error -- Yes --> Retryable{Is error retryable}
  Retryable -- No --> Fail[Raise error]
  Retryable -- Yes --> CSOTmode{Is CSOT enabled}
  CSOTmode -- Yes --> TimeLeft{Now before deadline}
  TimeLeft -- Yes --> Op
  TimeLeft -- No --> Fail
  CSOTmode -- No --> IsWrite{Is operation a write}
  IsWrite -- No --> Op
  IsWrite -- Yes --> RetryCount{Attempts less than two}
  RetryCount -- Yes --> Op
  RetryCount -- No --> Fail
```
----
### Prompt (With Chaining): "How should I implement CSOT timeouts in an algorithm? | Generate a mermaid diagram for CSOT timeouts. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
    Start[Start Operation] --> Init[Record start time and user timeouts]
    Init --> ComputeRemaining[Compute elapsed and remaining total]
    ComputeRemaining --> CheckTotal{Remaining total greater than zero}
    CheckTotal -- No --> Timeout[Abort with CSOT timeout error]
    CheckTotal -- Yes --> PhaseCheck{In connect phase}
    PhaseCheck -- Yes --> ComputePhaseConnect[phase timeout equals smaller of remaining total and connect timeout]
    PhaseCheck -- No --> ComputePhaseOther[phase timeout equals remaining total]
    ComputePhaseConnect --> Attempt[Perform operation attempt with hard deadline now plus phase timeout]
    ComputePhaseOther --> Attempt
    Attempt --> ResultCheck{Success or non retryable error}
    ResultCheck -- Yes --> Return[Return operation result]
    ResultCheck -- No --> ComputeRemaining
```
----
### Prompt (With Chaining): "How should I implement primary election in an algorithm? | Generate a mermaid diagram for primary election. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TD
  Start[Initialize state lastElectionId lastSetVersion lastProtocolVersion maxWireVersion]
  Start --> Discover[Discover new primary]
  Discover --> Extract[Extract electionId setVersion protocolVersion maxWireVersion]
  Extract --> CheckProto{lastProtocolVersion set and new protocolVersion not equal lastProtocolVersion}
  CheckProto -->|Yes| Reset[Accept new primary reset state values] --> End[End]
  CheckProto -->|No| CheckWire{maxWireVersion less than 17}
  CheckWire -->|Yes| MakeTuplesPre6[Form newTuple as newSetVersion newElectionId and oldTuple as lastSetVersion lastElectionId]
  CheckWire -->|No| MakeTuples6[Form newTuple as newElectionId newSetVersion and oldTuple as lastElectionId lastSetVersion]
  MakeTuplesPre6 --> Compare{newTuple less than oldTuple}
  MakeTuples6 --> Compare
  Compare -->|Yes| Stale[Mark member unknown keep old primary] --> End
  Compare -->|No| Accept[Accept new primary update state values] --> End
```
----
### Prompt (With Chaining): "How should I implement checking out a connection from a pool in an algorithm? | Generate a mermaid diagram for checking out a connection from a pool. The content of your diagram should not contain any quotation marks or parentheses."
```mermaid
flowchart TB
  Start --> SetupTimeoutTimer
  SetupTimeoutTimer --> CheckPoolState
  CheckPoolState -->|PoolClosed or PoolPaused| EmitError
  CheckPoolState -->|PoolActive| EnqueueCheckoutRequest
  EnqueueCheckoutRequest --> WaitQueueFront
  WaitQueueFront -->|NotAtFront| WaitQueueFront
  WaitQueueFront -->|AtFront| CheckAvailableConnections
  CheckAvailableConnections -->|AvailableCount greater zero| UseAvailableConnection
  CheckAvailableConnections -->|NoAvailable| CheckConnectionLimits
  CheckConnectionLimits -->|UnderMaxLimits| InitiateConnectionCreation
  CheckConnectionLimits -->|AtMaxLimits| AwaitCheckInOrCountChange
  InitiateConnectionCreation --> ConnectionCreationComplete
  ConnectionCreationComplete --> UseNewConnection
  UseAvailableConnection --> PrepareCheckedOutConnection
  UseNewConnection --> PrepareCheckedOutConnection
  AwaitCheckInOrCountChange --> WaitQueueFront
  PrepareCheckedOutConnection --> ReturnConnection
  EmitError --> End
  ReturnConnection --> End
```