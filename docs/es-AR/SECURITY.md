# Política de Seguridad

[English](../../SECURITY.md) |
[Português (Brasil)](../pt-BR/SECURITY.md)

La versión en inglés es la versión canónica de esta política.

## Versiones soportadas

Molejo Testkit es un proyecto experimental en pre-release. No existe una versión
soportada o lista para producción. Las correcciones de seguridad se aplican a la
branch predeterminada según la disponibilidad de los mantenedores.

## Uso previsto

Testkit es una fixture determinista de pruebas. No es una aplicación autenticada,
un servicio de autorización ni un backend de producción. Desplegalo únicamente
donde corresponda una carga de prueba y eliminá los entornos descartables después
de la validación.

El servidor HTTP público no expone intencionalmente el probe de egreso. El probe
es un comando explícito del contenedor destinado a Jobs controlados y diagnósticos
locales. Quien pueda ejecutar ese comando obtiene la identidad de red del
contenedor o Pod; por lo tanto, el acceso a la creación y ejecución de cargas sigue
siendo un límite de seguridad del cluster.

## Reportar una vulnerabilidad

El reporte privado de vulnerabilidades de GitHub todavía no está habilitado en
este repositorio. Hasta que exista un canal privado, contactá a los mantenedores
mediante el [perfil de la organización Molejo Platform](https://github.com/molejo-platform)
para solicitar un canal privado sin incluir detalles de la vulnerabilidad en el
mensaje inicial. No divulgues la vulnerabilidad en una issue, discussion, pull
request o log de prueba público.

Incluí, cuando sea posible:

- la revisión y el componente afectados;
- pasos para reproducir o una prueba de concepto mínima;
- el impacto de seguridad esperado;
- supuestos relevantes del despliegue;
- cualquier mitigación conocida.

Los mantenedores confirmarán y evaluarán los reportes según su disponibilidad.
Como el proyecto está en pre-release, actualmente no existe un SLA de respuesta o
corrección.

## Divulgación

Permití que los mantenedores tengan una oportunidad razonable para investigar y
preparar una corrección antes de la divulgación pública. El crédito se coordinará
con quien realizó el reporte cuando sea apropiado y deseado.
