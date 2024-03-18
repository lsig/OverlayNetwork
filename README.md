# OverlayNetwork

# Running the program

As we structured the codebase as a single go module, we couldn't find a way to have both `registry.go` and `messages.go` be executable from the root directory. We raised this issue up with Marcel on March 18th, and we discussed the possibility of the programs being executed using:

```go
go run registry/registry.go
```

```go
go run messages/messages.go <host>:<port>
```

To run the bash script which spins up 10 instances of messaging nodes and 1 registry instance you first need to:

chmod +x run.sh

Then simply run the script like so:

./run.sh


# Implementation details

A concern we raised with Marcel was that we saw that once all message nodes had sent their originating packets and sent a TaskFinished message to the registry, some packets were still in circulation in the network, being relayed between nodes. While this wasn't a problem for lower values of _n_, for larger ones, such as 100.000, the possibility of any packets being in circulation while all nodes had successfully delivered their packets was much higher.

We proposed a possible solution with Marcel, that we should consider relayed packets as having a higher priority than packets originating at the sending node. This would ensure that nodes try to relay packets before sending their own, hopefully resulting in the nodes sending the _TaskFinished_ message to the registry relatively later on in the process. Marcel did not _not_ like this solution, but thought that it would perhaps not help meaningfully. At the very least, would it not remove the problem altogether.

During the in-class discussion on this problem on March 18th, many students said that they simply got around this problem by inserting a sleep call at the registry after having received the last _TaskFinished_ packet, allowing all relaying packets to be delivered to their destinations. While being not as cool, its hard to argue with the effectiveness of this solution. Therefore, you'll find that the registry waits 5 seconds after receiving the last _TaskFinished_ packet and before casting _RequestTrafficSummary_ packets.

<br>

Finally, we decided that once all the message nodes had sent their _TrafficSummary_ messages, that they would gracefully shut down, closing all connections and stop listening as well, before terminating.
=======
# Work methodology

When we started working on the project, we decided that Logi would be responsible for the [registry.go](./registry/registry.go) part of the code, and Kristófer would be responsible for the [messages.go](./messages/messages.go) part. This worked well, as long as we were working on the same functionality at the same time (we met up at RU to work together on the code), as the message nodes and registry are so intertwined.

However, we realized that everytime that one of us would implement some functionality, it would benefit the other person to immediately get access to that functionality, as it would help them progress further on their code. Therefore, we quickly decided to scrap our methodology of working on our own separate branches, and would rather just commit to the main branch.

# Acknowledgements

## ChatGPT

Like all other students, we used AI tools like ChatGPT to help get us started by setting the project up, testing different implementations and help with some details. However, we don't rely on those tools, and only use it as a sort of search engine on steroids, while still independently making decisions on and programming the actual project.

Here are links to our ChatGPT threads:

[Logi's chat](https://chat.openai.com/share/dd30e84f-4cf0-4f95-9960-32acdf8903c5)

[Kristófer's chat](https://chat.openai.com/share/e9d11afe-72de-4886-832f-30e1319ba59b)

## Other internet threads and documentation

Below are links with explanations and examples of how to solve certain problems that we encountered while building this project. We didn't copy the code from these examples, but rather used the examples to see which packages and methods were useful for solving the tasks we got stuck on.

[How to access command line arguments](https://stackoverflow.com/questions/2707434/how-to-access-command-line-arguments-passed-to-a-go-program)

[Convert string to integer](https://stackoverflow.com/questions/4278430/convert-string-to-integer-type-in-go)

[net.ParseIp can be used to check validity of ip representation in string](https://stackoverflow.com/questions/19882961/go-golang-check-ip-address-in-range)

net.Dial, net.DialTCP, net.Listen and net.ResolveTCPAddr were used with help from [the official go _net_ docs](https://pkg.go.dev/net)

[channels](https://go.dev/tour/concurrency/2)

[sort list of complex objects](https://yourbasic.org/golang/how-to-sort-in-go/)
