# Build COmmands

### Build Container
```bash
podman build --tag registry.ohrenpirat.de:5000/mwildt/lasttesttest:latest -f containerfile .
```

### Run Container
```bash
podman run -p 9081:8081 -p 9082:8082 registry.ohrenpirat.de:5000/mwildt/lasttesttest:latest
```

### Push
```bash
podman push registry.ohrenpirat.de:5000/mwildt/lasttesttest:latest 
```


### Build + Push
```bash
podman build --tag registry.ohrenpirat.de:5000/mwildt/lasttesttest:latest -f containerfile .
podman push registry.ohrenpirat.de:5000/mwildt/lasttesttest:latest
```
