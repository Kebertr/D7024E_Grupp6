
When you want to lookup where a node/contact is located kadmelia does that by measuring the XOR distance from the target. The smaller the XOR distance is the closer the target is to the point of measure. First the node looks at its own routing table to find nodes that are closest to the target, so assume if k = 4 then it would look something like this:

Node        XOR distance
        ----        ------------
        A              10
        B              17
        C              25
        D              40


These four nodes are then made into a list called shortList that is sorted and returned by `func (routingTable *RoutingTable) FindClosestContacts(target *KademliaID, count int) []Contact` that is located in routingtable.go. A important claification here is that these nodes are not necessarily all located in the same bucket since there could be nodes closer to the target that are outside of the bucket.

When we have our shortList we start to query the closest nodes by simple asking every node in the shortList "Who do you know that is close to the target" which is the Remote Procedure Call (RPC) in Kademlia.

Then depending on the nodes awnsers we replace nodes with a known further distance to the target with nodes that are further away in our shortList while mainting a size of k. Repeat this procedure until there are no unqueried candidates left. So after every query you try to move closer to the target where we eventually end up in a situation where every node we have contacted either is farther away from the target node or does not know anyone closer to the target and it is here we have converged.



The alpha parameter determines how many nodes we can query concurrently so instead of asking one node and waiting for it to finish untill we start asking the next node, we can query through and alpha amount of nodes at the same time. So basically it is just how many requests we can send out simultaneously.

So having a k value of 10 and alpha of 3 means that we can maintain a shortList of up to 10 nodes, but only have 3 oustanding network requests at a time.

Helper functions: 

```go
func nextUnqueried(candidates []Contact, queried map[string]bool, alpha int) []Contact {
```

NextUnqueried selects the next group of contacts that have not been queried yet where: 


* `candidates []Contact`: possible contacts to query.
* `queried map[string]bool`: tracks contacts already queried, using each contact’s ID as the key.
* `alpha int`: maximum number of contacts to return.

Where it marks a contact as queried immediately when adding it to the returned batch so a later call will not return the same contact again.



```go
func queryBatch(ctx context.Context, network *Network, batch []Contact, targetID *KademliaID) ([][]Contact, error) {
```


queryBatch sends a lookup request to every contact in batch concurrrently and collects their responses so we can collect the results.

* `network`: sends the actual `FindContact` requests.
* `batch`: contacts to query.
* `targetID`: ID being searched for.

It does this by first prepering the communication channels where it creates one result slot for each contact. Then it starts one goroutine per contact where each is queried concurrently by sending a FindContact request to that contact. If successful the returned contacts and original index is sent back so all responses can be collected in the final for loop: 



```go
func mergeClosest(
	candidates []Contact,
	newContact Contact,
	targetID *KademliaID,
	count int,
) []Contact
```

mergeCloseset add a newly discovered contact to the candidate list while keeping only the closest contacts to the target. 


* `candidates`: current list of possible contacts.
* `newContact`: newly discovered contact.
* `targetID`: ID being searched for.
* `count`: maximum number of contacts to keep, usually `bucketSize`.
