# Common Issues

This guide covers common issues that you might encounter when using the AI-Enhanced Telemetry Processor, along with their solutions.

## Startup Issues

### Processor Fails to Start

**Symptoms**:
- The processor exits immediately after starting
- Error messages in logs about configuration or initialization

**Possible Causes and Solutions**:

1. **Invalid Configuration**:
   - Error messages like `invalid configuration: models.error_classifier.path is required`
   - **Solution**: Check your configuration YAML for syntax errors or missing required fields

2. **WASM Model Issues**:
   - Error messages like `failed to load WASM model: file not found`
   - **Solution**: Verify that the model paths are correct and the files exist

3. **Memory Allocation Failures**:
   - Error messages like `failed to allocate memory for WASM runtime`
   - **Solution**: Reduce memory limits or ensure the host has enough available memory

4. **Port Conflicts**:
   - Error messages like `failed to bind to address: address already in use`
   - **Solution**: Change the port configuration or stop other services using those ports

### WASM Model Load Failures

**Symptoms**:
- Error messages about WASM compilation or instantiation
- Specific error messages about model functions not being found

**Possible Causes and Solutions**:

1. **Corrupt or Invalid WASM Files**:
   - Error messages like `invalid magic number` or `failed to compile module`
   - **Solution**: Rebuild or redownload the WASM models

2. **Missing Export Functions**:
   - Error messages like `function 'classify_error' not found in module`
   - **Solution**: Ensure the WASM models implement the required functions

3. **Memory Constraints**:
   - Error messages like `memory allocation failed`
   - **Solution**: Increase the memory limit for the models

## Runtime Issues

### High CPU Usage

**Symptoms**:
- Processor consistently using more CPU than expected
- Slow telemetry processing

**Possible Causes and Solutions**:

1. **Too Many Concurrent Workers**:
   - **Solution**: Reduce the `concurrency` setting in the configuration

2. **Inefficient WASM Execution**:
   - **Solution**: Optimize the WASM models or increase their timeout values

3. **High Telemetry Volume**:
   - **Solution**: Implement pre-filtering or increase batch size

### High Memory Usage

**Symptoms**:
- Processor using more memory than expected
- Out of memory errors

**Possible Causes and Solutions**:

1. **Cache Size Too Large**:
   - **Solution**: Reduce the cache size for models

2. **WASM Memory Limits Too High**:
   - **Solution**: Reduce the memory limit for each model

3. **Queue Size Too Large**:
   - **Solution**: Reduce the queue size in the processing configuration

### Telemetry Not Being Processed

**Symptoms**:
- Telemetry is received but not enriched or processed
- No error messages, but no visible results

**Possible Causes and Solutions**:

1. **Feature Toggles Disabled**:
   - **Solution**: Check that the appropriate features are enabled in the configuration

2. **Sampling Rate Too Low**:
   - **Solution**: Increase the sampling rates in the configuration

3. **WASM Models Timing Out**:
   - **Solution**: Check logs for timeout errors and increase timeout values

### Data Loss or Drop

**Symptoms**:
- Missing telemetry data
- Error messages about queue overflow

**Possible Causes and Solutions**:

1. **Queue Size Too Small**:
   - **Solution**: Increase the queue size in the configuration

2. **Processing Too Slow**:
   - **Solution**: Increase concurrency or batch size

3. **WASM Models Too Slow**:
   - **Solution**: Optimize the models or increase their timeout values

## Model-Specific Issues

### Error Classifier Issues

**Symptoms**:
- Errors not being classified correctly
- Missing `ai.error.*` attributes in telemetry

**Possible Causes and Solutions**:

1. **Pattern Matching Failures**:
   - **Solution**: Update the error classifier model with additional patterns

2. **Timeout Issues**:
   - **Solution**: Increase the timeout value for the error classifier model

3. **Memory Constraints**:
   - **Solution**: Increase the memory limit for the error classifier model

### Importance Sampler Issues

**Symptoms**:
- Too much telemetry being sampled out
- Important telemetry being missed

**Possible Causes and Solutions**:

1. **Sampling Rates Too Aggressive**:
   - **Solution**: Adjust the sampling configuration

2. **Model Not Identifying Important Events**:
   - **Solution**: Update the importance sampler model to better recognize key patterns

3. **Threshold Issues**:
   - **Solution**: Adjust the `threshold_ms` value for slow span detection

### Entity Extractor Issues

**Symptoms**:
- Missing or incorrect entity information
- Empty entity arrays in output

**Possible Causes and Solutions**:

1. **Pattern Matching Failures**:
   - **Solution**: Update the entity extractor model with additional patterns

2. **Input Data Format Issues**:
   - **Solution**: Ensure telemetry data has the expected format and attributes

3. **Memory or Timeout Constraints**:
   - **Solution**: Increase memory limit or timeout for the entity extractor model

## Configuration Issues

### Environment Variable Override Problems

**Symptoms**:
- Configuration not being applied as expected
- Default values being used despite overrides

**Possible Causes and Solutions**:

1. **Incorrect Variable Names**:
   - **Solution**: Verify the exact names of environment variables

2. **Precedence Issues**:
   - **Solution**: Understand that file-based configuration takes precedence over environment variables

3. **Type Conversion Issues**:
   - **Solution**: Ensure environment variables use the correct format for their type

### Pipeline Configuration Issues

**Symptoms**:
- Processor not receiving telemetry
- Processed telemetry not reaching exporters

**Possible Causes and Solutions**:

1. **Missing Pipeline Links**:
   - **Solution**: Verify that the processor is included in all desired pipelines

2. **Order Issues**:
   - **Solution**: Check the order of processors in the pipeline

3. **Exporter Configuration**:
   - **Solution**: Verify that exporters are correctly configured and working

## Advanced Troubleshooting

### Enabling Debug Logging

To get more detailed logs:

```yaml
service:
  telemetry:
    logs:
      level: "debug"
```

### Health Check Endpoint

Monitor the health of the processor:

```bash
curl http://localhost:13133
```

### Metrics Endpoint

Check processor metrics:

```bash
curl http://localhost:8888/metrics
```

### Profiling

For performance issues, enable the pprof endpoint:

```yaml
extensions:
  pprof:
    endpoint: 0.0.0.0:1777
```

Then use the Go pprof tools:

```bash
go tool pprof http://localhost:1777/debug/pprof/profile
```

### Core Dumps

For serious crashes, enable core dumps (Linux):

```bash
ulimit -c unlimited
```

## Getting Help

If you're still experiencing issues:

1. Check the project [GitHub repository](https://github.com/fortxun/caza-otel-ai-processor/issues) for similar issues
2. Submit a detailed bug report with your configuration and logs
3. Join the OpenTelemetry community discussions on Slack or GitHub