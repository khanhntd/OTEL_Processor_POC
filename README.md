# OTEL_Processor_POC

# Output:
```
2022-08-12T10:43:39.601-0400    info    awsemfexporter@v0.57.2/emf_exporter.go:184      Finish processing resource metrics      {"kind": "exporter", "data_type": "metrics", "name": "awsemf", "labels": {"InstanceId":"testInstanceId"}}
```

Sadly, for `awsemfexporter` does not recognize at Resource Attributes level but only at Data Point Attributes level. Therefore, only able to recognize the change in console.
