# Some thinking about improvements and new features

		sorry for MEIBIO

 [-] Firstly need to switch from http communication to [go rpc](https://pkg.go.dev/net/rpc?tab=doc), it must be most faster and more suitable for localhost communication. I planned use rpc other unix sockets.

 [-] Make class name for block changable. Block has name and class name. Class name affects on css styling. It's possible to changing class name by scripting for changable appearance due to state of the block.

 [-] Make nested blocks. Each block can has a parrent block. Parent's block css style affects on style of children blocks. Like nested divs in HTML. By default all blocks is child of the `bar`.
