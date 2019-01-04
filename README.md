# elrond-go-sandbox

The package `p2p` is used for testing libP2P and other P2P related issues.

TODO list:
1. make sure a malicious peer that sends empty blocks for each round, do not break the current node 

#####Example of starting bootnode:

You need to run in .../eldond-go-sandbox/cmd/bootnode folder:
 
./go build<br />
./go install

Then you have to put in one {FOLDER} the files bellow:
 
{FOLDER}/bootnode (binary generated by above commands)<br />
{FOLDER}/config/config.testnet.json<br />
{FOLDER}/genesis.json

With the text below could be created a start.sh file in the same 
{FOLDER} and run it. This is all! 

```
# Start n instances in a group of <consensusGroupSize> (genesis.json) from a total of <initialNodes> (genesis.json) validators,<br />
# each of them having maximum -max-allowed-peers (flag) peers, with round duration of <roundDuration> (genesis.json) miliseconds<br />
# in sync mode at the subround level <elasticSubrounds> = false (genesis.json) (with <elasticSubrounds> = true time of solving the consensus will decrease)

# Private keys:

#  1: ZBis8aK5I66x1hwD+fE8sIw2nwQR5EBlTM8EiAOLZwE=
#  2: unkVM1J1JvlNFqY3uo/CvAay6BsIL3IzDH9GDgmfUAA=
#  3: Or0C7+gvlr/kIZLS+tiBBQfbUQ+pqS9FTE3dXfs5Swg=
#  4: 0i3nK1VRtHEGKXkejwcU9JYfbtO/WwvapxF8qLfX/Ag=
#  5: lKUBBHFuqwbKLy4xJshK8I/WND3JcaHZ+P1Pk9W8YAg=
#  6: jSNFvjEnds2JvB9v74l5oBbsCsFZNxuvkwjUh1m+xQw=
#  7: HbfcRATSr697pGqawbQIllutzK1ChTUGB+BD+ZpOPAs=
#  8: p57gu5OHtDRIWmgwNqm0k77XIi73KUywHAJfuAExtwk=
#  9: QhNuOhB7/9MymO9izhC43x5aiwI4NPSfjfeSxfj8BwY=
# 10: 5WCLn8HHsEvu0NE51dPCuLPVfssd005Y4trYshr6sAc=
# 11: te4x8jjWXIyjLB77zRgcmNR4NBpZXkeVVcpGKoRo/wo=
# 12: BEBCSKoB2gBUj0+AZvs9sxFIe0rMkxRrNFNHt/fg7gk=
# 13: dE+/RIIP+UFC9RX+rAmwosjrjBIO16q07dqvCvp44ww=
# 14: T9KhsEpTlNEmpNbh8KJkIwFufYKnwxHoj8Si+Hf2ww4=
# 15: JaTuf9jrXhsGnVmWWxOFaa4IZG8bqGwp/RnZ2QV9nww=
# 16: rN4LSVhJQ4sfvgTilJ0yozErT0NEI3/TUZZVPzhYrAo=
# 17: UoazF8yMgjPQwJ4jYkw4hwjhmTOFMBXXUhLEOBQ/1Ak=
# 18: RLs0DJOwwxAX+yMA3Vyu3MRtPA/CTClKovMNfVsl9QA=
# 19: FYXMafB0++EsBK2F1X5dpdmNvau6l72jrJnH9zcYOAE=
# 20: //Cq4pEA6SW6NZeJtq4xzK/5JuZlblFbeyAPpl5/KgU=
# 21: fJFUv3dbGZuupDnTc22q5XRXCNLl1lEmmStyrguXwg0=

# Public keys:

#  1: bCYAUf+qhQtYKFfgQ1g3JstkJFVTsA2KAH+0L+qZlO4=
#  2: gDI39ZN3loP1Cujru6+BJtu+gNwQnBB8g4yVW0wyuaA=
#  3: TLkPlhd8g07tiqE4Mgvq1kCp3EOEEjn8O3/DpyjUqFE=
#  4: YO0S6tNhNjWwsJJzTLrJMecEKDLeuZlJNznf7nU/TcM=
#  5: Gkfqv+PTR9ot2TILrOBFcPojmhwE9IC7Y2psLc9ZsZc=
#  6: nVnGbEbxPR4ab0thyeV/O1FZopNDTdNexNI5OPCGtRo=
#  7: xMonbskDZ1dHeW4vh3/AUzf7psbPIPTKfGz+J6gmpeA=
#  8: 6kVjlTw6NmSqV4kI6cEprxd+2f37FXCzpXnFsYYcsLU=
#  9: 0NNr5LuHjgSSiXFKp37uPAjGt1HYdQJTUmJ4ASKGZyQ=
# 10: JyK0pKNF3UnzIFtm9pVDXF3xArpfhrU4g2bbGxwHBwE=
# 11: bT4UtBU3A5MpKWvvpL5nAplLAGEWccoJb9NzdXJN1fk=
# 12: S+goExeC5GNLPx9xYnhXB8mVQQzGJpr0B1QCU/DqqdM=
# 13: 1KGnod6xGDML+y5qNiwrYq2t2M1Elgbt2YNfXLRSow0=
# 14: 84uQq3tJFjzSZ6KIcJwb7JZ1yLwuDytcPHYITVlnNjA=
# 15: 8pqG75sKgeqIKim2jR/P7ojM5QgSQkHLt6xpablZoM0=
# 16: E8u9qxcr8hQ3nM6RfOOLS4bzu9fV+whiTtOY5kjaDlE=
# 17: M+C7UJoK6poGvlPqkLVpFOuWao9dFtCoHVWaWv0ee3U=
# 18: NBu1klLKrye76DblD8IhexsHrai2TD4+8KdWBWWqaxc=
# 19: SBoD6sJA5oEmyhwK5CckOlH3ByrJkyJZyih+iy1tGno=
# 20: NUhC12eqQ0U5IIjpytuaWzBSSPYE2myNVro8I/3Rjpg=
# 21: +cmFvcxUN9roSQnW8AMpi3kq0LDSTZCIrL8f7Pc6ez4=

gnome-terminal -- ./bootnode -port 4000 -max-allowed-peers 4 -private-key "ZBis8aK5I66x1hwD+fE8sIw2nwQR5EBlTM8EiAOLZwE="
gnome-terminal -- ./bootnode -port 4001 -max-allowed-peers 4 -private-key "unkVM1J1JvlNFqY3uo/CvAay6BsIL3IzDH9GDgmfUAA="
gnome-terminal -- ./bootnode -port 4002 -max-allowed-peers 4 -private-key "Or0C7+gvlr/kIZLS+tiBBQfbUQ+pqS9FTE3dXfs5Swg="
```