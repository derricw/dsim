# dsim

lightweight process simulation

### define a process model

Process are defined in yaml.

```yaml

# Example Model

# Processes are the nodes of our system. Here we
# define the names of each node of our system that produce
# or consume.
processes:
    producer:
        out:
            gizmos: 4
            goobers: 2
        duration: 10m
        replicas: 2
    consumer:
        in:
            gizmos: 8
            goobers: 4
        out:
            widgets: 1
        duration: 2m
        replicas: 1


# How many (free-floating) copies of each
# widget are allowed to exist without systems
# waiting? If undefined here, it is assumed to be 1000.
max_pool_size:
    gizmos: 100
    goobers: 50
    widgets: 1000

```

### run the process

Simulate your process for 3 hours.

```bash
dsim run for 3h process.yml
```

Output:

```
Process	   Completed	InProgress	
-------	   ---------	----------
consumer   17		    1		
producer   36		    2	

Pool	Unbatched
----	---------
gizmos	0
goobers	0
widgets	17
```

### build

```
make build
```

### install

```
make install
```

### roadmap

* `run until` command to run until a condition is met
* `optimize` command to pick optimal replica counts from a matrix

