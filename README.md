# OverlayNetwork

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
