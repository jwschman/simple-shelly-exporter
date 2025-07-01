# simple-shelly-exporter

A *very* simple Prometheus exporter for my Shelly Plug US.  At the moment it only works for a single plug, and only exposes the active power (watts) metric.  But that's all I wanted in the first place.

## Setup

### 1: Start simple-shelly-exporter container

The program can be run on its own in Docker (or whatever container engine you prefer).  Personally I run it as a pod inside a local Kubernetes cluster ([my manifests here](kubernetes-manifests/readme.md)).

#### Environment Variables

| Name | Description | Example value |
| --- | --- | --- |
| `SHELLY_PASSWORD` | Password used for Shelly authorization.  Leave blank if none set | password (or blank) |
| `SHELLY_HOST` | Hostname or IP address the Shelly plug | 192.168.1.1 |
| `LISTEN_PORT` | Port used to serve metrics | 2112 |

### 2: Add Prometheus scrape job

If necessary, add the scrape job to your `prometheus.yml` configuration:

```yaml
scrape_configs:
- job_name: myapp
  scrape_interval: 10s
  static_configs:
  - targets:
    - localhost:2112 # change to the host simple-shelly-exporter is running on
```

In my case, since simple-shelly-exporter is running in the same Kubernetes cluster as Prometheus, I set the target to `simple-shelly-exporter.monitoring.svc.cluster.local:2112`

### 3: Use the metrics however you want

That's it.  You're good to go 😎

## TODO

- Expose additional metrics from the Shelly Plug
- Support multiple plugs (I only have one, so...)

## About

I've had a Shelly Plug US sitting around for a year and finally decided to start using it while I'm checking the power usage for some new hardware.  Since I'm already using Prometheus inside my homelab environment I thought it would work well in there. 

After briefly looking for shelly exporters I thought it would be a good project to write my own.  I did initially find [this one here](https://github.com/geerlingguy/shelly-plug-prometheus) but it seems to be using an old version of the Shelly API that my device doesn't use.

After reading the documentation for writing Go applications to serve Prometheus metrics I wrote this little guy mostly as a coding exercise.  It does exactly what I wanted it to do, and nothing more at the moment.  

## References

- [Instrumenting a Go application for Prometheus](https://prometheus.io/docs/guides/go-application/)
- [Golang Prometheus Package Documentation](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus)
- [shelly-plug-prometheus](https://github.com/geerlingguy/shelly-plug-prometheus)

## License

All software in this repo is released to the public domain under the [unlicense](https://unlicense.org/).
