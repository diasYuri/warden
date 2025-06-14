Segue uma **task** detalhada, no nível de um Staff Engineer, para projetar e implementar o **componente de State Machine** como uma biblioteca Go reutilizável:

---

## Task: Criar Biblioteca Go de State Machine para Workflow Engine

### 1. Objetivo

Desenvolver uma **state machine** genérica em Go, modular e configurável, que possa ser embutida no Core de Workflow Engine. A biblioteca deve permitir modelar, persistir e transicionar estados de forma segura, observável e extensível.

### 2. Contexto

O Workflow Engine orquestra tarefas definidas pelo usuário via um grafo de nós (tasks, decisões, paralelismo). A state machine é responsável por:

* Representar o grafo de estados e transições.
* Gerir o ciclo de vida de cada instância (inicialização, execução, falhas, conclusão).
* Persistir estado e eventos para garantir **resiliência** e **replay** em caso de falhas.
* Emitir *hooks* de eventos para monitoramento e retry.

### 3. Requisitos Funcionais

1. **Modelagem de Estados**

   * Definir tipos de estados: `Start`, `Task`, `Decision`, `Parallel`, `Join`, `End`.
   * Permitir extensão com novos tipos de estado via *interface*.

2. **Transições**

   * Registrar transições nomeadas (ex.: `OnSuccess`, `OnFailure`, `OnTimeout`).
   * Validar no build-time (compile) e runtime se toda transição referenciada existe.

3. **Instância de Execução**

   * Criação de instância em estado inicial.
   * API para avançar estado: `Next(event Event)`, `Goto(stateID string)`.
   * Expor métodos para rollback ou compensação em caso de falha.

4. **Eventos e Listeners**

   * Publicar eventos de estado: `StateEntered`, `StateExited`, `TransitionFired`.
   * Permitir registro de *listeners* ou *hooks* (e.g. para métricas, logs, armazenamento).

5. **Persistência**

   * Interface para persistir e carregar instâncias (ex.: via PostgreSQL JSONB ou Redis).
   * Versão mínima de esquema para garantir compatibilidade (`version` no modelo).
   * Suporte a snapshot + replay de eventos (event sourcing) para reconstruir estado.

6. **Conformidade com Concorrência**

   * Suportar safe‑concurrency: múltiplas goroutines podem avançar instâncias isoladas.
   * Bloqueio granular para evitar race conditions (e.g. mutex por instância).

7. **Idempotência**

   * Cada transição acionada mais de uma vez deve ter efeito **único**.
   * Gerenciar chaves de idempotência para eventos repetidos.

### 4. Requisitos Não‑Funcionais

* **Performance**: Latência de transição ≤ 5 ms por operação em média (medido em ambiente de teste).
* **Escalabilidade**: Suportar ≥ 50k instâncias ativas simultâneas; footprint de memória controlado.
* **Observabilidade**: Emissão de spans OpenTelemetry em cada transição; logs estruturados (zap/logrus).
* **Testabilidade**:

  * Cobertura ≥ 90% em unit tests.
  * Testes de integração simulando replay de eventos, falhas e concorrência.
* **Documentação**:

  * Godoc completo para todas as APIs públicas.
  * Guia de “Getting Started” e exemplos de uso (configuração, persistência, listeners).

### 5. Design e Padrões de Projeto

* **State Pattern**: cada estado é uma estrutura que implementa interface `State` (métodos `Enter`, `Execute`, `Exit`).

* **Observer Pattern**: para *listeners* de eventos; use canal ou pub/sub interno.

* **Builder / Fluent API**: para definição de máquina em código, e.g.:

  ```go
  sm := fsm.New("order-process").
      State("start").
      State("charge", fsm.Task(...)).
      State("notify", fsm.Task(...)).
      Decision("approved", condition).
      Transition("start", "charge", fsm.OnSuccess).
      // ...
      Build()
  ```

* **Event Sourcing**: armazenar eventos em append-only log e snapshots periódicos.

### 6. Entregáveis

1. **Código Go**

   * Módulo `github.com/yourorg/fsm-engine` (versão semântica).
   * Arquivos principais:

     * `state.go` (definição de interface e estados)
     * `machine.go` (core da state machine)
     * `persistence.go` (interfaces e implementações de persistência)
     * `listener.go` (pub/sub interno)
     * `examples/` (exemplos de criação e uso)

2. **Testes**

   * Suite de unit tests e mocks de persistência.
   * Testes de integração em memória e com Redis/Postgres (usando Docker Compose).

3. **Documentação**

   * Godoc gerado.
   * `README.md` com visão geral, API reference e exemplos.

4. **Benchmark & Relatório de Performance**

   * Benchmarks com `go test -bench` para latência de transição e throughput.
   * Resultados e configuração do ambiente de teste.

### 7. Critérios de Aceitação

* Todos os testes passam com cobertura ≥ 90%.
* Demonstração de criação, persistência e replay de uma instância de workflow simples via exemplo.
* Latência de transição inferior a 5 ms em bench local.
* Demonstração de listeners gravando métricas via OpenTelemetry.
* Documentação clara e exemplos funcionando.

### 8. Cronograma Sugerido

| Atividade                             | Duração     |
| ------------------------------------- | ----------- |
| Design da API e modelagem de estados  | 2 dias      |
| Implementação core da state machine   | 5 dias      |
| Persistência & event sourcing         | 3 dias      |
| Listeners                             | 2 dias      |
| Testes e benchmarks                   | 3 dias      |
| Documentação e exemplos               | 2 dias      |
| **Total estimado**                    | **17 dias** |

---

> Com esta task, um Staff Engineer tem um escopo completo, desde design até entrega, garantindo que o componente de **state machine** se comporte de forma confiável, performática e extensível dentro do Workflow Engine.
