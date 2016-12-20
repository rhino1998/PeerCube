

###Constants
* M: Size of identifiers
* N: 2^M, The max size of entire network
* S<sub>min</sub>: Minimum size of a cluster
* S<sub>max</sub>: Maximum size of a cluster. S<sub>max</sub> > M

## Functions

D(a,b) c {

return Σ<sub>i=0, a<sub>i</sub>≠b<sub>i</sub></sub><sup>m-1</sup>2<sup>m-i</sup>

} 


## Starting State
Complete routing tables with perfect hypercube structure
This state is not changed by the following operations

## Cluster
* **Dim**: length of label and RT
* **Label**: Dim-bit number that prefixes contained peers
* **RT[i]**: The i-th element of RT contains the Cluster that differs from Label in the i-th bit
* **Vc**: Core members of the cluster. Each member is prefixed by label


## Peer
Internally refered to as *p*
* Attributes
    * **ID**: m-bit identification number
    * **Cluster**: The cluster to which the peer belongs
    * **Type**: Core, Spare, or Temp
    * **Data**: Key-Value store

* Methods
    ```
    Lookup(key){
        Register a lookup response handler for key
        if p.Type != "Core"{
            Send ("LOOKUP", key, p, p) to (Smin-1)/3+1 random peers in p.Cluster.Vc
            Val <- Wait for responses
        }else{
            C <- Closest Cluster to key by distance D
            Send ("LOOKUP", key, p, p) to (Smin-1)/3+1 random peers in C.Vc
            Val <- Wait for responses
        }
        Clear the lookup response handler for key
        return Val
    }
    ```
    ```
    Put(key,val){
        time <- system.time
        Register a put response handler for key
        if p.Type != "Core"{
            Send ("PUT", key, val, time, p, p) to (Smin-1)/3+1 random peers in p.Cluster.Vc
            Val <- Wait for responses
        }else{
            C <- Closest Cluster to key by distance D
            Send ("PUT", key, val, time, p, p) to (Smin-1)/3+1 random peers in C.Vc
            Val <- Wait for responses
        }
        Clear the lookup response handler for key
        return Val
    }
    ```

* Message Reactions
    ```
    ("LOOKUP", key, origin, prev)  
        C <- Closest Cluster to key by distance D
        if p.cluster = C{
            if lookup response handler for key not registered{
                Register lookup response handler for key
                Broadcast ("Lookup", key, origin, p) to p.Cluster.Vc
                Val <- Wait for responses
                If p.Data[key] = Val {
                    Send ("LOOKUPRETURN", key, Val) to prev
                }else if p.Data[key] = nil{
                    p.Data[key] <- Val
                    Send ("LOOKUPRETURN", key, Val) to prev
                }else{
                    Send ("LOOKUPRETURN", key, nil) to prev
                }
                Clear the lookup response handler for key
            }else{
                Send ("LOOKUPRETURN", key, p.Data[key]) to prev
            }
        }else{
            Register a lookup response handler for key
            Send ("LOOKUP", key, origin, prev) to (Smin-1)/3+1 random peers in p.Cluster.Vc
            Val <- Wait for responses
            Send ("LOOKUPRETURN", key, Val) to prev
            Clear the lookup response handler for key
        }
    ```
    ```
        ("LOOKUPRETURN", key, Val){
            Verify Val with lookup response handler for key
        }
    ```
    ```
        ("PUT", key, val, time, origin, prev){
            if p.cluster = C{
            if a put response handler for key is not registered{
                Register a put response handler for key
                Broadcast ("PUT", key, val, time, origin, p) to p.Cluster.Vc
                p.Data[key, time] = val //If existing value time is newer, do nothing
                Wait for responses
                Send ("PUTRETURN", key) to prev
                Clear the put response handler for key
            }else{
                p.Data[key, time] = val //If existing value time is newer, do nothing
                Send ("PUTRETURN", key) to prev
            }
        }else{
            Register a put response handler for key
            Send ("PUT", key, val, time, origin, prev) to (Smin-1)/3+1 random peers in p.Cluster.Vc
            Wait for responses
            Send ("PUTRETURN", key) to prev
            Clear the put response handler for key
        }
    ```
    ```
        ("PUTRETURN", key){
            Acknowledge response with the put response handler for key
        }
    ```