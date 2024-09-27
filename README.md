# snuffle
Rules-based Network Threat Intelligence proof-of-concept

# Overview
Observability is absolutely crucial in modern systems.  The days of running `gdb` against running processes, or interrogarting `/proc`, or parsing log files are long behind us - the advent of distributed cloud systems, and the sheer volumes of data involved, make these methods utterly impractical and unsuitable.

In the modern world, "*Observability*" describes the ability to take an end-to-end view of a system, zooming in and out to arbitrary levels of granularity, in order to diagnose problems or tune performance or detect security violations.

The available tooling has come a long way.

In the beginning, there was Intrusion Detection Systems (IDS), such as `Snort`.

Then, Security Incident and Event Management (SIEM) became much more popular, such as `Splunk`.

Now, Extended Detection and Response (XDR) is becoming more prevalent, such as `Wazuh`.

At the same time, the available data for telemetry and observability has come a long way (`PCAP` and `SNMP` has given way to things like `eBPF` and `GNMI`) and it is idiomatic nowadays for telemetry to be baked into application code with things like `Prometheus` and `OpenTelemetry`.

I'm not happy simply to be able to operate the tools of others (e.g. Wazuh) - I want to know how they work at a fundamental level.  I want to know what the tradeoffs are, where the scaling challenges and performance bottlenecks can be, what the limitations are and some of the ways of addressing them.  The best way for me to do that is to develop an implementation of my own.

I recall writing an operating system from scratch (Xinu) so that I could fully understand Unix and Linux.  I recall writing my own compiler, to understand optimisation and code generation.  I recall writing my own JVM, so I could fully understand Java bytecode and become very proficient at performance tuning because I knew the internals of garbage collection etc.

So I developed `Snuffle` as part of taking a deep dive into all of the above, to get a thorough understanding of all the the technologies involved and how a modern Observability stack fits together.

If you're looking for an out-of-the-box XDR, I'd recommend `Wazuh` - it's a fantastic product.

If you're looking to understand Observability, and how to implement an Observability platform in Go, then you'll get an awful lot out of `Snuffle`.

# Architecture
`Snuffle` is intentionally a self-contained binary.  This is mainly to avoid writing a lot of transport code (e.g. protobufs and gRPC) which ultimately have nothing to do with the moving parts of Observability and XDR.

At the same time, the `collector` package is the single ingress for all events and data, so it is reasonably trivial to extend that to expose a network ingress and allow data to be captured from remote systems if I chose to build that functionality.

## Topology
```
+-------------+
| Sources     |
+-------------+
| - PCAP      |
| - eBPF      |
| - Telemetry |
| - GNMI      |
| - Snuffle   |
+------+------+
       | [SnuffleEvent]
       V
 +-----+-----+      +--------+      +-------+       +-+----------+
 | Collector +----->+ Filter +----->+ Rules +---+--->+ Reporting |
 +-----------+      +--------+      +-------+   |   +--+---------+
                                                |
                                                V   +---+--------+
                                                +-->+  Actions   |
                                                    +----+-------+            
```

Each of these components is described below.

Something that might instantly catch your eye is that `Snuffle` can also be a `Source`.  This makes it composable and chainable, the huge advantages of which are fully explained in the `Collector` and `Actions` sections below.

It should also be fairly obvious to you that the `Rules` component is going to be the most computationally expensive, and therefore the most likely bottleneck that will benefit from tuning and optimisation.  I am not going to try to optimise this prematurely but, at the same time, I am going to make absolutely sure that initial design decisions will support - and not prevent or restrict - any optimisation that may be needed.  This is a key factor in the early decision to allow `Snuffle` output to also be a `Source`.

An absolute truism in optimisation and performance is: "*The best optimisation is to find something you're doing that doesn't need doing, and don't do it*".  `Snuffle` has been built with this axiom in mind.

### SnuffleEvent
A `SnuffleEvent` is the common data structure used throughout `Snuffle`, provided to the `Collector` by a `Source`.

As well as the payload of the event, the `SnuffleEvent` also carries any necessary metadata about the event (e.g. device, processId, user, timestamp, timezone, language etc).

For the initial implementation it is a simple struct.  By creating it as a first-class type, this allows it to be refactored to be things like:

  - JSON
  - Protobuf
  - XML
  - any other serialisable format

to support future refactoring if we wanted to make `Snuffle` a distributed system of agents and queues, and perhaps even a chain of collectors (aka a "*Collector Sieve*") to allow "cheaper" rules to run before "expensive" rules in a way that is embarassingly parallel and will therefore scale well.

Such a "*Collector Sieve*" also supports the ability to have different rules for different situations (or different organisations / device types / etc) - again without the need for significant refactoring.

### Sources
The various sources of data and events, located in the `source/` package.  Think of these as being the external stimuli or sensors which collect raw data.  They may or may not do some initial filtering.

Sources *push* `SnuffleEvent` to `Snuffle`.  Specifically, there is nothing built into `Snuffle` at this stage where it does any polling.

It is the responsibility of a `Source` to provide `Snuffle` with the data in the format / structure that `Snuffle` requires.  This avoids a lot of potentially byzantine logic inside `Snuffle` having to build lots of `if...else...` logic to deal with data that has come from different sources, since such logic all over the code can be a major source of bugs.

A `Source`, at its heart, is essentially its own ETL implementation:

  - Extract (or obtain) data from an external thing
  - Transform it into the `Snuffle` data structure / format
  - Load / ingest it (i.e. pass it to the `Collector`)

### Collector
The `Collector` is the sole ingress point for `Snuffle`, and is located in the `collector/` package.

The initial implementation is an internal function call, taking the data as arguments.  Specifically, there is no intervening external queue (e.g. Kafka) and the "queue" is simply a channel of events.

The reason for this, again, is to concentrate on the functional aspects of `Snuffle` without getting bogged down by the plumbing implementation.  At the same time, the `Collector` is architected in a way that an external queue (or multiple queues) could easily be slotted in.

Because the `Collector` only receives `SnuffleEvent`, it does not know or care how that event came into being.  Whilst an event can certainly come from some kind of sensor, it is also possible for the event to come from another `Snuffle` instance.

This supports a chain of `Snuffle`, each of which processes `Rules` that become increasingly expensive:

```
  +------------------+     +------------------------+
  | Snuffle          +---->+ Snuffle                +----> ...
  | (Cheapest rules) |     | (More expensive rules) |
  +------------------+     +------------------------+
```

By "*expensive*", I am also including the fact that the cost of executing each rule is added to the cost of all of the non-matching rules which ran before.  So when I use the word "*expensive*", you could easily substitute the word "*common*" or "*likely*" or "*relevant*".  

It might even be the case that some instances of `Snuffle` run on very expensive hardware (e.g. GPU machines if AI is involved, or things that have specific emulators which can process platform-specific code or something) and we obviously only want to pay for that hardware to process the tings we absolutely know require it - dealing with everything else on much cheaper hardware.

More succinctly, we want to run the rules that are most likely to match, and are the cheapest, as early as possible.  We only want to run esoteric or expensive rules in the event that the simpler / quicker / cheaper rules don't match.

For instance, here are two rules (and how common they are):

  - `TCP_Port_80` (matches 80% of the time)
  - `Content_Type_JSON` (matches 10% of the time)

Clearly, we'd want to execute them in that order, because there's a 0.8 probability of getting a match on the first rule.  In other words: "*as much as possible, do not execute rules that you don't have to*".

Of course, this could also be done in code, without system boundaries, through clever rules ordering.  But allowing cascading / chaining in this way brings other significant benefits.

Imagine that there are 1,000 rules.  The "cost" of each rule (ie how likely it is to match) is simply its number (1, 2, ..., 1000).  Simple ordering of the rules, in a single `Snuffle` will easily take care of the imperative "*do the cheapest rules first*".

Now imagine that we have events from two different customers (or subnets, or some other separate and distinct thing).

Events from subnet1/customer1 will _only_ be matched by odd-numbered rules.  Events from subnet2/customer2 will _only_ be matched by even-numbered rules.

When an event from subnet2/customer2 is received, then we can say with certainty:

  - best case: we execute 1 rule that we absolutely did not have to because it is completely irrelevent (i.e. rule1 only applies to customer1)

  - worst case: we execute 500 rules that we absolutely did not have to because it is completely irrelevent (i.e. rule(1, 3, 5, ..., 999) only applies to customer1)

That's an awful lot of wasted compute and an awful lot more wasted wallclock time.

By having a `Collector` dedicated to customer1 (rules [1, 3, 5, ..., 999]) and a separate `Collector` dedicated to customer2 (rules [2, 4, 6, ..., 1000]), our best and worst case scenarios for a customer2 event now become:

  - best case: we execute 0 rules that we absolutely did not have to

  - worst case: we execute 0 rules that we absolutely did not have to

Which is significantly more preferable, and utterly in keeping with our axiom of "*don't do things that you don't have to do*".

The job of the `Rules` component is to "*perform desired actions, based on specified criteria*".  This distinction is important when you consider the role of the `Filter` component.

### Filter
In keeping with our mantra of not doing things that don't need to be done, the job of the `Filter` is to discard - as early and as cheaply as possible - any `SnuffleEvent` that would just be wasting the time of the rules enging because it's never going to match.

So the job of the `Filter` component, found in the `filter/` package, is to "*Keep things away from the rules engine that we 100% know do not have to be evaluated*".

Arguably, this could also be done with rules.  For example:

  - `tcp port 80 && ipv4` ==> DISCARD

But that slightly lower up-front effort comes at the cost of constraining us in the future.

For instance, the above `DISCARD` might be the tenth rule (for some reason), which is wasteful if we (pointlessly) always execute the 9 rules that precede it.

Some of the job of the `Filter` might even be undertaken by each `Source` - i.e. discard irrelevant things as early as possible.  But that then creates a whole new set of requirements around configuring each source, propagating the configurations, keeping them all in sync (etc) - all of which is orthogonal to the XDR proof-of-concept that I'm building with `Snuffle`.

For a `Source` of PCAP or eBPF, filtering can be done extremely cheaply at source and it is idiomatic to do as much filtering at the edge as possible (notwithstanding the above new requirements that consequently get invited to the party).  In a production-grade system, this filtering-at-the-edge would definitely be built in.

The key thing to note is that there needs to exist some concept of a `Filter` insofar as "*only pass things to the rules engine that we cannot 100% predict is guaranteed to match nothing*".

Consider, for example, if there were misbehaving devices (or some kind of fault) on a subnet which is resulting in a huge event storm that is totally contaminating all of the XDR (in some arbitrary way).  We might want to squelch all events from that subnet until the problem is resolved.  Or we might want to squelch events emitted by devices from a specific vendor, or a specific application or something.

For the initial implementation, this will be a NOP.  But, for an absolutely tiny up-front cost of implementing the NOP (with the correct interface) and placing it in the correct place in the processing pipeline, we've given ourselves a huge amount of flexibility to deal with things in a future we cannot predict or foresee.

Which is a favourite mantra of mine: "*You can't see round corners, but don't paint yourself into one*".

### Rules
The `Rules` component, found in the `rules/` package, does what it says on the tin.

For each event:

  - that is received by the `Collector`
  - which passes the `Filter`
  - evaluate it against the configured rules
  - if a matching rule is found:
    - report the matching rule
    - perform the associated actions (if any)

In other words, it's pretty much the same heuristic performed by any rules engine anywhere in the world.

Implicit in the above logic is a "*stop on first matching rule*".  This is for two reasons:

  - it's significantly easier to implement
  - it's not clear that a complex DAG of rules and logic is needed at this point (YAGNI principle)

It's pretty obvious at this early stage that the rules engine will be the beating heart of `Snuffle`.  At the same time, I don't want to over-engineer and over-abstract the rules engine to meet requirements that I don't have yet.

To mitigate this, `Rules` is a simple interface:

```
  type RulesEngine interface {
    Evaluate(ctx Context, event *SnuffleEvent) (Rule, error)
    Run(ctx Context, event *SnuffleEvent) (Rule, error)
  }
```

The first of these evaluates the rules, and indicates which rule (if any) matches the event - *but does not execute any associated actions (including reporting)*.

The second one does the same evaluation, but also invokes any associated actions (including reporting).

Without getting too hung up at an early stage of what the rules engine will look like, or what it's grammar / DSL / config will be, it gives the ability to get the entire XDR pipeline implemented to allow rapid testing and iteration.

In keeping with the composable design of `Snuffle`, any compliant implementation of the above interface can be slotted in.  So the rules engine could be an external microservice, a complex gramar, or anything inbetween.

What does it mean to "*perform associated actions*"?  In order to answer that question, we need to understand the `Actions` component.

### Actions
The `Actions` component, found in the `actions/` package, is responsible for executing any actions required when a rule is matched.

Consider the following rule (expressed in simple YAML, purely for readability - I have not yet made a decision on how the rules and actions will be expressed in config):

```
  - rules
    - TcpPort80
      criteria: tcp && port 80
      actions:
        - Do_Something_Web_Server
```

It's pretty obvious what's going on here.  Expressed in pseudologic, it's even simpler:

```
  if event.proto is TCP and event.port == 80:
    Do_Something_Web_Server(event)
```

The job of the `Actions` component is to execute these actions, using some kind of priority queue.  It does not know (or care) why it is being asked to perform actions, it simply executes the next one in the queue - looping again and again.

The actions are completely arbitrary.  It might be a DNS lookup, sending an alert, or something else.  It is not the job of the `Actions` component to know what it is being asked to do, its job is simply to reliably do it.  Any and all circuit-breaking / retry logic is all delegated to (and contained in) the `Actions` module.

As has been described above about the cascading of `Snuffle` instances, there is a special action called `Snuffle` - in other words, pass this event to a new `Snuffle` instance.  Again, as a simple concrete example using YAML:

```
  - rules
    - TcpPort80
      criteria: tcp && port 80
      actions:
        - Snuffle ${OTHER_SNUFFLE_IDENTIFIER}
```

where `${OTHER_SNUFFLE_IDENTIFIER}` is a `IP:PORT`, or a kafka queue or something else - that's not important.  The important part is that there is a specific action which simply passes the entire `SnuffleEvent` to another `Snuffle` instance to be dealt with.  It might be another in-memory instance, it might be a remote system, it might be a queue or something else - the actual implementation is not important.

What is important is that the ability to cascade `Snuffle`, to build scalable tiers and scalable pipelines, has been baked in from the very beginning for almost zero effort - and without making things so abstract that they are impossible to get your head around.

This very simple, and very cheap to implement early, design decision opens up such a huge amount of flexibility that (in my opinion) it far outweighs the YAGNI imperative.

### Reporting
No XDR system, or Observability platform, is useful without the ability to report on what is going on.

The `Reporting` module, located in the `reporting/` package provides this.

For this first implementation, all it does is log the following for a `SnuffleEvent`:

  - the rule that matched (and, optionally, why)
  - there was no matching rule

Again, this functionality is contained behind a single interface:

```
type Reporter interface {
    Report(event *SnuffleEvent, matched *Rule) error
}
```

(obviously, if `matched` is `nil`, then this means "*the event matched no rule*").

This encapsulation allows many degrees of freedom if we ever wanted to hook up to any other external reporting / audit system in the world, because the invocation has been baked in at the correct stage of the pipeline - we simply need to implement the communication that we need.




