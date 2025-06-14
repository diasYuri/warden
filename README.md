# Warden - Go State Machine Library

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-007d9c.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Warden é uma biblioteca Go para criar e gerenciar máquinas de estado (state machines) de forma reutilizável, performática e observável. Projetada para ser o core de workflow engines, oferece recursos como persistência, event sourcing, paralelismo e monitoramento.

## ✨ Características

- **🏗️ Builder Pattern**: API fluente para construção de máquinas de estado
- **🔄 Tipos de Estado**: Start, Task, Decision, Parallel, Join, End
- **💾 Persistência**: Suporte a snapshots e event sourcing
- **📊 Observabilidade**: Listeners para métricas, logs e eventos
- **⚡ Performance**: Latência < 5ms por transição
- **🔒 Thread-Safe**: Concorrência segura com bloqueios granulares
- **🔁 Idempotência**: Controle de eventos duplicados
- **🎯 Extensível**: Interfaces para novos tipos de estado

## 🚀 Instalação

```bash
go get github.com/diasYuri/warden/pkg
```

## 📖 Uso Básico

### Criando uma Máquina de Estado Simples

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/diasYuri/warden/pkg"
)

func main() {
    // Constrói a máquina de estados
    machine, err := warden.New("order-processing").
        StartState("start").
        TaskState("validate", validateOrder).
        TaskState("process", processOrder).
        EndState("completed").
        EndState("failed").
        Transition("start", "validate", warden.TransitionOnSuccess).
        Transition("validate", "process", warden.TransitionOnSuccess).
        Transition("validate", "failed", warden.TransitionOnFailure).
        Transition("process", "completed", warden.TransitionOnSuccess).
        Transition("process", "failed", warden.TransitionOnFailure).
        Build()
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Cria uma instância
    instance, err := machine.CreateInstance("order-123", map[string]interface{}{
        "order_id": "ORD-123",
        "amount":   99.99,
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Executa o workflow
    ctx := context.Background()
    err = instance.Start(ctx)
    
    if err != nil {
        log.Printf("Error: %v", err)
    }
    
    fmt.Printf("Final state: %s\n", instance.CurrentStateID)
    fmt.Printf("Completed: %v\n", instance.IsCompleted())
}

func validateOrder(ctx warden.StateContext) warden.StateResult {
    // Lógica de validação
    return warden.StateResult{
        Success: true,
        Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
    }
}

func processOrder(ctx warden.StateContext) warden.StateResult {
    // Lógica de processamento
    return warden.StateResult{
        Success: true,
        Event:   warden.NewEvent(warden.TransitionOnSuccess, nil),
    }
}
```

## 🏗️ Tipos de Estado

### 1. StartState - Estado Inicial
```go
machine.StartState("start")
```

### 2. TaskState - Estado de Tarefa
```go
machine.TaskState("my_task", func(ctx warden.StateContext) warden.StateResult {
    // Lógica da tarefa
    return warden.StateResult{
        Success: true,
        Data: map[string]interface{}{"result": "done"},
        Event: warden.NewEvent(warden.TransitionOnSuccess, nil),
    }
})
```

### 3. DecisionState - Estado de Decisão
```go
machine.DecisionState("check_amount", func(ctx warden.StateContext) (warden.TransitionType, error) {
    amount := ctx.Data["amount"].(float64)
    if amount > 100 {
        return warden.TransitionOnSuccess, nil // Premium
    }
    return warden.TransitionOnFailure, nil // Standard
})
```

### 4. ParallelState - Estado Paralelo
```go
machine.ParallelState("parallel_tasks", []func(warden.StateContext) warden.StateResult{
    task1,
    task2,
    task3,
})
```

### 5. JoinState - Estado de Junção
```go
machine.JoinState("wait_all", []string{"task1_done", "task2_done"})
```

### 6. EndState - Estado Final
```go
machine.EndState("completed")
```

## 💾 Persistência

### Persistência em Memória
```go
persister := warden.NewInMemoryInstancePersister()
machine := warden.New("my-workflow").
    WithPersister(persister).
    // ... states and transitions
    Build()
```

### Event Sourcing
```go
eventPersister := warden.NewInMemoryEventPersister()
persister := warden.NewEventSourcePersister(eventPersister)
machine := warden.New("my-workflow").
    WithPersister(persister).
    // ... states and transitions
    Build()
```

### Carregando Instâncias
```go
// Carrega uma instância existente
instance, err := machine.LoadInstance("instance-123")
if err != nil {
    log.Fatal(err)
}

// Continua execução
ctx := context.Background()
err = instance.Start(ctx)
```

## 📊 Observabilidade

### Event Bus e Listeners
```go
eventBus := warden.NewEventBus()

// Listener de métricas
metricsListener := warden.NewMetricsListener()
eventBus.Subscribe(metricsListener)

// Listener de logging
loggingListener := warden.NewLoggingListener(warden.LogLevelInfo)
eventBus.Subscribe(loggingListener)

machine := warden.New("my-workflow").
    WithEventBus(eventBus).
    // ... states and transitions
    Build()
```

### Métricas
```go
metrics := metricsListener.GetMetrics()
for metric, count := range metrics {
    fmt.Printf("%s: %d\n", metric, count)
}
```

## 🔧 Configurações Avançadas

### Configuração Personalizada
```go
config := &warden.StateMachineConfig{
    EnablePersistence:      true,
    EnableEventBus:         true,
    MaxConcurrentInstances: 5000,
    TransitionTimeout:      30 * time.Second,
    RetryAttempts:          3,
    RetryDelay:             1 * time.Second,
}

machine := warden.New("my-workflow").
    WithConfig(config).
    // ... states and transitions
    Build()
```

## 🎯 Controle de Fluxo

### Transições Manuais
```go
// Força transição para um estado específico
err := instance.Goto(ctx, "specific_state")

// Envia evento personalizado
event := warden.NewEvent(warden.TransitionOnCustom, map[string]interface{}{
    "reason": "manual_intervention",
})
err := instance.Next(ctx, event)
```

### Verificação de Estado
```go
// Estado atual
currentState := instance.GetCurrentState()

// Transições disponíveis
transitions := instance.GetAvailableTransitions()

// Status da instância
if instance.IsRunning() {
    fmt.Println("Instance is running")
}
if instance.IsCompleted() {
    fmt.Println("Instance completed successfully")
}
if instance.IsFailed() {
    fmt.Printf("Instance failed: %v", instance.Error)
}
```

## 📁 Estrutura do Projeto

```
warden/
├── pkg/                   # Código da biblioteca
│   ├── state.go          # Definições de estado e interfaces
│   ├── machine.go        # Core da máquina de estados
│   ├── persistence.go    # Interfaces de persistência
│   ├── listener.go       # Sistema de eventos
│   ├── errors.go         # Definições de erro
│   └── utils.go          # Funções utilitárias
├── tests/                # Testes
│   └── warden_test.go    # Suite completa de testes
├── examples/             # Exemplos de uso
│   ├── basic_workflow.go # Exemplo básico
│   └── advanced/
│       └── advanced_workflow.go # Exemplo avançado
├── go.mod               # Dependências
├── go.sum               # Checksums
├── README.md            # Esta documentação
└── requirement.md       # Especificação original
```

## 🧪 Executando Testes

### Todos os Testes
```bash
go test ./tests/ -v
```

### Benchmarks
```bash
go test ./tests/ -bench=. -benchmem
```

## 🧪 Executando Exemplos

### Exemplo Básico
```bash
cd examples
go run basic_workflow.go
```

### Exemplo Avançado
```bash
cd examples/advanced
go run advanced_workflow.go
```

## 📈 Performance

- **Latência**: ~14μs por transição (muito abaixo dos 5ms requisitados)
- **Throughput**: > 80,000 operações/segundo
- **Memória**: ~6.5KB por transição, footprint controlado

## 🔒 Thread Safety

A biblioteca é completamente thread-safe:
- Locks granulares por instância
- Estados imutáveis após construção
- Operações atômicas em estruturas compartilhadas

## 🤝 Idempotência

Controle automático de idempotência:
```go
// Eventos duplicados são automaticamente detectados
event := warden.NewEvent(warden.TransitionOnSuccess, nil)
err1 := instance.Next(ctx, event) // Sucesso
err2 := instance.Next(ctx, event) // Retorna ErrDuplicateEvent
```

## 🧪 Cobertura de Testes

A biblioteca possui cobertura completa de testes incluindo:

- ✅ Testes unitários para todos os componentes
- ✅ Testes de integração 
- ✅ Testes de concorrência
- ✅ Testes de persistência
- ✅ Testes de event sourcing
- ✅ Testes de performance
- ✅ Testes de idempotência
- ✅ Testes de validação

## 📝 Licença

Este projeto está licenciado sob a Licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🚧 Roadmap

- [ ] Persistência em PostgreSQL
- [ ] Persistência em Redis
- [ ] Integração com OpenTelemetry
- [ ] Retry policies configuráveis
- [ ] Timeout por estado
- [ ] Webhooks para eventos
- [ ] Dashboard web para monitoramento

## 👥 Contribuição

Contribuições são bem-vindas! Por favor, leia as diretrizes de contribuição antes de submeter PRs.

## 📞 Suporte

Para questões e suporte, abra uma issue no GitHub.

---

**Warden** - Construindo workflows confiáveis e observáveis em Go 🚀 