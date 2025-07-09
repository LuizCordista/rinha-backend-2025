Implementação do backend da rinha de maneira assíncrona

## Sobre a Solução

Este projeto foi feito para a Rinha de Backend 2025, buscando equilibrar desempenho, resiliência e o menor custo possível por transação. O backend recebe os pagamentos e faz o processamento de forma assíncrona, sempre tentando usar o Payment Processor default (que tem a menor taxa) e só recorrendo ao fallback quando realmente necessário.

### Como funciona
- Os pagamentos recebidos pelo endpoint `/payments` entram em uma fila no Redis e são processados em background, o que permite receber muitas requisições sem travar o sistema.
- O serviço acompanha o health check dos Payment Processors, que busca os estados a cada 5 segundos, e escolhe o melhor serviço para continuar a requisição.
- Todos os pagamentos processados ficam registrados no próprio Redis para garantir que o resumo retornado pelo `/payments-summary` esteja sempre alinhado com os dados dos Payment Processors.
- O backend roda com pelo menos duas instâncias, como pede o desafio, e pode ser escalado se precisar.

## Resultados dos testes

![image](https://github.com/user-attachments/assets/308570d8-7886-4fbf-a5c4-b1b044d48f76)
