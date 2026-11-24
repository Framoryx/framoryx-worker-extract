# RFC 0001 Execução de extrações e persistência

Estado: Parcialmente confirmado para o primeiro incremento do domínio. Data: 21/11/2026.

O plano delimita o contrato de execução entre o serviço de negócio em Spring e o worker Go, antes de conectar mensageria e processamento real. A direção proposta mantém os metadados e tentativas sob controle do serviço e atribui ao worker o processamento e o relato dos resultados. O primeiro incremento implementa a modelagem dos modos e trechos escolhidos; a etapa seguinte deverá testar a orquestração com dependências simuladas.

Após o merge e a autorização de continuidade, o primeiro incremento aplica D4/D6: resultado lógico único e validação de saída com frames e tamanho de ZIP positivos. Para D5 foram confirmados intervalo configurável e positivo, ordem cronológica e rejeição de sobreposições. O contrato de domínio exige modo explícito e ao menos um trecho [Start, End), com início incluído e fim excluído; trechos adjacentes são válidos. A validação não ordena nem une seleções. O modo de todos os frames exige intervalo de amostragem zero, pois esse parâmetro não se aplica a ele.

Esta confirmação não fecha contratos de transporte, limites operacionais, recuperação ou integrações das etapas seguintes. A representação com `time.Duration` não promete precisão de FFmpeg nem decide como quadros serão mapeados para timestamps.

## Decisões para avaliação

| Decisão | Recomendação e alternativa | Consequência e etapa bloqueada |
|---|---|---|
| D1 Propriedade dos dados | Serviço Spring persiste metadados e transições; alternativa de dois escritores não recomendada | Mantém consistente o bloqueio de expiração manual com trabalhos pendentes; necessária antes da coordenação |
| D2 Retorno de execução | HTTP com consulta e repetição idempotentes; eventos de resultado como alternativa | HTTP depende da disponibilidade do serviço; eventos exigem recuperação durável da publicação e consumo idempotente |
| D3 Identidade de tentativa | Serviço atribui a tentativa na aquisição; worker gera somente uma chave estável para repetir essa aquisição | Duplicatas de entrega não viram novas tentativas; necessária antes das portas de coordenação |
| D4 Unidade de resultado | Confirmado para o incremento: um ZIP por solicitação com todos os trechos | Modelo mantém resultado lógico singular; a unicidade oficial por job depende da persistência/coordenação da etapa 3; geração do ZIP ainda será implementada |
| D5 Amostragem e todos os frames | Confirmados modos explícitos, frequência configurável e positiva, ordem cronológica e ausência de sobreposição | Implementados na validação do domínio; precisão operacional e limites de produto permanecem pendentes |
| D6 Critério de sucesso | Confirmado para o incremento: rejeitar zero frames e ZIP com tamanho zero | Implementado na validação de resultado; não prova existência ou integridade física do objeto |
| D7 Confirmação do broker | Inicialmente após resultado ou falha definitiva aceitos duravelmente; aquisição com ACK antecipado é alternativa condicionada a recuperação durável | Exige compatibilidade entre tempo do trabalho e janela de entrega; necessária antes do consumidor real |
| D8 Recuperação e repetição | Separar recuperação de execução interrompida de repetição de falha; definir classificação, limite de tentativas e destino de mensagens inválidas | A retomada de trabalhos aceitos precisa de garantia durável; novas tentativas automáticas de falhas continuam pendentes |
| D9 Referência dos arquivos | Original fixado e destino exclusivo/recuperável por job e tentativa; alternativa depende de controle equivalente | Evita processar conteúdo alterado e sobrescrever resultado oficial; necessária antes do armazenamento |

PNG/JPEG continuam explícitos. Aliases ou formato padrão só precisam ser decididos se o contrato permitir omitir/normalizar o formato. Podem continuar adiados nesta avaliação: valores de timeout/lease e backoff, capacidade, limites de duração/resolução, qualidade de imagem, nomes de buckets, TTL de temporários e hospedagem. Cada valor precisa ser definido antes da respectiva integração; adiamento não significa capacidade ilimitada ou autorização para defaults arbitrários.

## Contexto e escopo

O worker já possui [domínio de extração](../../internal/domain/extraction/), [testes públicos](../../tests/domain/extraction/) e infraestrutura de observabilidade. O [bootstrap](../../cmd/main.go) ainda não consome trabalhos. Não há implementação de coordenação, armazenamento, extração ou persistência de negócio.

`ExtractionOptions` distingue amostragem (`sampling`) e todos os frames (`all_frames`) com seleção explícita de trechos. `FrameInterval` é positivo no primeiro modo e zero no segundo. `VideoSegment.Validate` verifica início não negativo e fim posterior ao início; a seleção exige ordem cronológica e ausência de sobreposição. `ExtractionResult.Validate` exige quantidade de frames e tamanho de ZIP positivos, mantendo duração de processamento não negativa.

O domínio verifica a estrutura dos trechos, sem consultar a duração real do vídeo; a compatibilidade com a mídia de origem será validada na execução. As demais seções continuam propondo responsabilidades, garantias e sequência de implementação, sem alterar dependências ou infraestrutura. A aprovação de uma direção arquitetural não define automaticamente os parâmetros de produto que aparecem como pendentes neste RFC.

A grade de amostragem, incluindo se reinicia no início de cada trecho ou segue uma referência global, permanece pendente para a etapa 4. Um intervalo maior que o trecho continua representável; o executor deverá aplicar a convenção definida e verificar se há quadros selecionáveis. O início incluído é uma fronteira temporal, não a garantia de existir um quadro exatamente nesse timestamp. Sem frames, o resultado não pode ser aceito como concluído.

As seções de responsabilidades, persistência e coordenação descrevem um contrato entre aplicações; a ordem de implementação delimita o trabalho interno do worker e a contraparte necessária. A localização em `docs/rfcs/` é provisória: a organização oficial da documentação e o repositório do serviço Spring ainda precisam ser definidos. Publicar aqui permite revisar o contrato junto ao código existente, sem determinar a quantidade de serviços nem substituir uma visão C4 do sistema.

## Responsabilidades propostas

| Informação ou operação | Serviço Spring responsável pelo recurso | Worker Go |
|---|---|---|
| Vídeo, solicitante, permissões e validade | Valida e persiste os metadados oficiais | Recebe somente os dados necessários à execução |
| Solicitação e parâmetros de extração | Aceita o trabalho e registra os parâmetros efetivos | Valida o contrato recebido e executa os parâmetros |
| Tentativa vigente | Autoriza, persiste e encerra a tentativa | Adquire a execução, renova a reserva e informa seu resultado |
| Original e ZIP | Autoriza referências e mantém seus vínculos com o recurso | Lê o original, gera frames/ZIP e envia o resultado ao armazenamento |
| Estado apresentado ao usuário | Aplica transições válidas e persiste o estado oficial | Reporta andamento e acontecimentos da execução |
| Recuperação e limpeza de órfãos | Identifica trabalhos abandonados e autoriza nova tentativa ou limpeza | Cancela trabalho sem autorização vigente e remove seus temporários locais |

O worker não escreve nas tabelas de negócio do serviço. A propriedade dos dados pertence ao serviço que aplica suas regras; essa separação é compatível com o padrão de [dados privados por serviço](https://docs.aws.amazon.com/prescriptive-guidance/latest/modernization-data-persistence/database-per-service.html). Não há necessidade proposta de banco próprio do worker nesta etapa.

Os módulos de vídeos e extrações podem começar no mesmo serviço Spring. Quantidade, nomes e repositórios desses serviços não são definidos por este RFC. Se forem separados posteriormente, será necessário detalhar também a consistência entre aceite da extração e ciclo de vida do vídeo.

## Identidade e referência dos arquivos

| Identificador ou referência | Semântica proposta |
|---|---|
| `videoId` | Identifica o original cadastrado, reutilizável em várias extrações |
| `jobId` | Identifica uma solicitação de extração e seu resultado lógico |
| `attemptId` | Identifica uma execução autorizada do job; uma recuperação cria outra tentativa |
| Chave de aquisição | Gerada pelo worker antes da chamada e mantida nas repetições da mesma aquisição; permite consultar sua resposta durável |
| `requestId` e IDs de tracing | Identificam chamadas e rastreabilidade; não substituem a identidade da tentativa de negócio |
| Referência do original | Bucket, key e versão fixada; sem versionamento, exigir uma chave final imutável |
| Referência do resultado | Destino exclusivo por job e tentativa, atribuído ou validado pelo serviço |

O serviço gera os IDs de negócio e controla a tentativa vigente. A entrega do broker não constitui uma nova tentativa por si só: mensagens duplicadas devem consultar o estado existente. A representação externa dos IDs permanece pendente; o domínio atual somente rejeita valores vazios ou em branco.

O aviso de despacho identifica o job, sem atribuir por conta própria uma nova tentativa. O worker monta `ExtractionJob` após adquirir a execução e receber `attemptId`, origem e parâmetros autorizados. A chave de aquisição não é uma credencial e não substitui a identificação do executor autenticado. A idempotência distingue consulta/repetição da mesma aquisição de autorização para outro executor assumir a reserva.

`tracecontext.NewAttempt`, usado na observabilidade existente, renova IDs de rastreabilidade; ele não cria nem autoriza uma tentativa de negócio. O contrato proposto não vincula `attemptId` ao ciclo de vida de um span ou requestId.

`VersionID` opcional no tipo Go não significa aceitar um original mutável. A política deve fixar a versão ou impedir sobrescrita da chave aceita. Resultados de tentativas diferentes não compartilham um destino sobrescrevível. Na mesma tentativa, repetir um upload não pode alterar um objeto já confirmado: reutilizar um artefato validado ou iniciar outra tentativa exige a autorização correspondente. Uma chave determinística auxilia localização e recuperação, mas não prova equivalência dos bytes. Os nomes concretos de buckets e prefixos serão definidos com o adapter de armazenamento.

## Momentos de persistência

| Momento | Registro durável sob responsabilidade do serviço |
|---|---|
| Solicitação de upload | Vídeo inicial, proprietário, sessão/referência prevista e estado aguardando upload |
| Upload concluído e aceito | Referência imutável confirmada, tamanho verificado, instante de aceite e validade do original |
| Aceite de extração | Job, vínculo com vídeo e solicitante, parâmetros, prazo solicitado para o resultado e intenção de despacho |
| Aquisição de execução | Tentativa vigente, autorização temporária de execução e destino exclusivo do resultado |
| Durante execução | Renovação da autorização e checkpoints que vierem a ser definidos; progresso visual é informação auxiliar |
| Conclusão aceita | Referência final do ZIP, medidas do resultado, encerramento da tentativa, instante de disponibilidade e validade do ZIP |
| Interrupção ou falha | Causa técnica classificada e situação da tentativa; o serviço decide se há nova tentativa ou falha definitiva |

O registro da solicitação antecede a existência física do ZIP. A disponibilidade começa somente após upload concluído, validação do resultado e aceite da tentativa vigente. A validade do original começa no upload concluído e aceito; a do ZIP começa quando a extração termina e o resultado fica disponível, independentemente da validade do original. Valores de retenção são regras do serviço, não parâmetros de execução do worker.

O aceite de um job e a intenção de publicá-lo devem fazer parte da mesma transação. Um publicador posterior envia essa intenção ao broker e registra a confirmação, tolerando repetição. O padrão [transactional outbox](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html) trata a janela entre commit no banco e envio da mensagem; não torna banco, broker e armazenamento uma única transação.

## Coordenação da execução

A recomendação inicial é um contrato interno de operações explícitas com resposta de aceite durável. O transporte candidato é HTTP. As operações abaixo descrevem semântica; não fixam rotas, payloads ou códigos de resposta.

| Operação | Garantia necessária |
|---|---|
| Adquirir tentativa | Reservar atomicamente um job elegível, atribuir tentativa e devolver referências/parâmetros autorizados; reconhecer duplicata ou job já concluído |
| Consultar execução | Permitir descobrir o resultado de uma chamada cuja resposta se perdeu, sem iniciar outra tentativa implicitamente |
| Renovar tentativa | Prorrogar somente a tentativa ainda vigente, usando o relógio do serviço como autoridade |
| Confirmar resultado | Validar identidade, referência e critérios de sucesso; persistir resultado e estado na mesma transação; repetir a mesma confirmação sem duplicar efeitos |
| Informar falha | Encerrar/classificar a tentativa válida sem converter automaticamente toda falha em falha definitiva do job |

A reserva temporária de execução, ou lease, precisa expirar e ser recuperável. A transação que aceita um resultado verifica a tentativa vigente e sua autorização. Uma confirmação antiga ou uma renovação atrasada não reativa uma tentativa substituída. O worker interrompe novas operações quando perde a autorização; a checagem no serviço protege o resultado mesmo se o processo antigo demorar a parar.

A chave de aquisição mantida apenas na memória protege repetições no mesmo processo, não sobrevive a reinício. Um novo executor consulta o estado do job e aguarda ou recupera a reserva conforme autorização do serviço; não tenta reconstruir a aquisição antiga a partir de requestId.

Repetir a mesma conclusão retorna o resultado já aceito, mesmo depois do vencimento da reserva; o reconhecimento desse resultado precede a exigência de autorização para uma nova transição. Uma conclusão com conteúdo diferente para a mesma tentativa deve ser rejeitada como conflito. Timestamps de disponibilidade, notificações e outras consequências não são repetidos por causa de reenvio.

Eventos de resultado são uma alternativa ao retorno por HTTP. Eles exigem publicação durável, confirmação do broker e consumo idempotente no serviço. Acrescentar uma outbox no serviço não resolve sozinho uma queda do worker antes de publicar seu resultado. A alternativa deve demonstrar recuperação dessa janela antes de substituir a coordenação proposta.

## Despacho e confirmação do broker

O produto de broker continua pendente. RabbitMQ aparece nesta seção somente como referência para comparar garantias e limites; sua versão e tipo de fila não foram escolhidos.

A recomendação inicial é confirmar consumo somente após o serviço aceitar duravelmente o resultado ou uma falha definitiva. Assim, a mensagem permanece disponível para repetição até existir um desfecho oficial. Quando uma duplicata chega e o job já terminou, o estado durável permite confirmá-la sem executar novamente.

Essa opção exige compatibilidade entre duração dos trabalhos e janela de entrega do broker. A definição adiada dos limites de duração/resolução e as medições do processamento bloqueiam essa verificação antes da etapa de consumo real. Caso RabbitMQ seja adotado, seu [timeout de confirmação de consumo](https://www.rabbitmq.com/docs/consumers#acknowledgement-timeout) deve ser verificado para a versão/tipo de fila escolhidos. Renovar o lease no serviço ou manter o heartbeat da conexão não deve ser tratado como extensão desse timeout de entrega.

Uma alternativa para trabalhos longos é tratar a mensagem como aviso e confirmar consumo após aquisição durável. Ela permanece indisponível até que a contraparte real implemente aquisição, recuperação de reservas expiradas e novo despacho duráveis e testados. O lease isoladamente não recupera um job depois que a mensagem foi removida. Não habilitar essa alternativa antecipada com apenas simuladores ou sem comprovar a recuperação.

Uma duplicata durante execução não cria outro processamento nem assume a reserva vigente. No modelo de confirmação ao final, seu tratamento aguarda um desfecho durável ou adia nova entrega conforme a política escolhida. Mensagem sem contrato válido exige rejeição/quarentena durável e rastreável; erro transitório não deve gerar um ciclo ilimitado de reenvios imediatos. A expiração de uma reserva não equivale à autorização de repetição ilimitada de falhas de processamento.

No exemplo do RabbitMQ, [publisher confirms e consumer acknowledgements](https://www.rabbitmq.com/docs/confirms#are-publisher-confirms-related-to-consumer-delivery-acknowledgements) cobrem relações diferentes com o broker. Nenhum deles substitui a gravação do estado oficial. Não se propõe garantia de execução exatamente uma vez: o objetivo é impedir múltiplos resultados oficiais e recuperar trabalhos aceitos.

## Falhas e recuperação

| Cenário | Comportamento proposto |
|---|---|
| Job salvo e envio ao broker falha | Manter intenção durável; repetir despacho com a mesma identidade do job |
| Aquisição foi aceita e resposta se perdeu | Consultar/repetir aquisição com identidade idempotente; não criar tentativa concorrente |
| Worker cai antes de concluir | Expirar a autorização; serviço agenda recuperação durável conforme política de tentativas |
| ZIP enviado e resposta de conclusão se perde | Consultar estado e repetir a mesma conclusão; não reexecutar enquanto o resultado durável puder ser confirmado |
| Worker cai após upload e antes de reportar | O serviço não declara sucesso pela mera presença do objeto; recupera a tentativa ou agenda nova tentativa e trata o objeto antigo como candidato a órfão |
| Tentativa antiga termina depois da substituta | Rejeitar conclusão antiga; manter o artefato oficial protegido contra sobrescrita |
| Banco/serviço indisponível na conclusão | Preservar a possibilidade de recuperar o trabalho e repetir a confirmação; falha de comunicação não prova falha do processamento |

Após reinício, memória e disco temporário do worker não são fontes de estado durável. Para reaproveitar um ZIP já enviado, a recuperação precisa de evidências completas e verificáveis, como referência imutável, identidade da tentativa e manifesto do resultado. Presença do objeto e tamanho isolados não comprovam todos os campos de `ExtractionResult`.

A proposta inicial admite reprocessar em outra tentativa quando essas evidências não existirem. A política de tentativas deve impedir abandono silencioso; o resultado antigo não pode ser tornado oficial após a substituição. Um checkpoint ou manifesto durável para reduzir reprocessamentos pode ser acrescentado depois, sem pressupor banco próprio do worker. A remoção de órfãos verifica tentativa vigente, resultado oficial e operações ainda em andamento antes de excluir objetos; suas janelas operacionais serão definidas com o armazenamento.

## Ordem proposta de implementação

Cada etapa abaixo pode originar um PR próprio após aprovação do seu recorte. Dependências simuladas verificam a orquestração; não representam integração real.

| Etapa | Entrega | Verificação esperada |
|---|---|---|
| 1 Entregue neste incremento | Domínio alinhado aos modos/trechos, unidade de resultado e critérios de sucesso de D4 a D6 | Testes table-driven de entradas válidas/inválidas, limites estruturais e compatibilidade com PNG/JPEG; a grade temporal do executor permanece pendente para a etapa 4 |
| 2 | Caso de uso Go com portas mínimas definidas pelo consumidor, sem adapters externos, após D1 a D3 e a etapa 1 | Testar ordem das operações, cancelamento, erros de cada etapa, limpeza e separação entre resultado produzido e confirmação aceita |
| 3 | Persistência/coordenação no serviço responsável, com operações idempotentes e recuperação | Testes de aquisição concorrente, expiração, renovação tardia, conclusão duplicada/conflitante e rejeição de tentativa antiga; sem transação aberta durante o processamento |
| 4 | Adapters de extração, arquivo, armazenamento e coordenação, em PRs por integração, após os contratos e D9 | Testes de contrato com fixture aprovada, origem fixada, extração/ZIP sem manter todo o resultado em memória e falhas entre upload e confirmação |
| 5 | Despacho durável, consumidor e ligação ao bootstrap, após a contraparte real, D7 e D8 | Injetar queda antes/depois da aquisição e da conclusão, redelivery, indisponibilidade, espera por tentativa vigente e recuperação conforme a política de ACK |

O caso de uso da etapa 2 deverá receber o contexto e distinguir aquisição, produção do artefato e confirmação do resultado. As portas descrevem necessidades do fluxo, sem importar SDKs ou conceitos de broker no domínio. A forma exata dos contratos será revisada junto à etapa 3, antes de qualquer cliente de coordenação real.

A etapa 3 pertence ao serviço responsável; sua implementação e repositório ainda não existem neste recorte. As etapas 2 e 4 podem usar simuladores para testes de contrato, mas a etapa 5 e a validação ponta a ponta exigem a contraparte real. Nenhum teste contra simulador comprova persistência, autenticação ou recuperação no serviço de produção.

Dois cuidados atravessam as etapas: autenticação e limites de acesso das integrações devem ser detalhados antes de expô-las; logs e métricas não podem substituir persistência ou conceder autorização. A instrumentação futura preserva o prefixo `framoryx.app`, sem dependência de telemetria no domínio.

## Critérios de validação da proposta

- Há uma autoridade identificada para cada estado durável e para a tentativa vigente.
- Toda solicitação aceita permanece recuperável, inclusive após encerramento do processo ou remoção da mensagem.
- Duplicatas e respostas perdidas não criam múltiplos resultados oficiais nem repetem efeitos de conclusão.
- Cada referência do original aponta para conteúdo fixado; uma tentativa obsoleta não sobrescreve o resultado oficial.
- A recuperação após upload sem confirmação não depende apenas de memória, logs ou temporários locais.
- Os modos de extração e critérios de sucesso são decididos antes de implementar suas regras; valores técnicos pendentes não são inventados.
- A primeira implementação tem um recorte testável com dependências simuladas; o domínio permanece dependente somente da biblioteca padrão.

A avaliação deve confirmar ou alterar as recomendações ainda pendentes da tabela de decisões. O estado parcialmente confirmado limita a autorização ao incremento descrito; o merge deste documento não autoriza automaticamente todas as implementações futuras.
