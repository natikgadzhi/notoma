{%- extends 'markdown.tpl' -%}


{% block body %}
---
{%- for k, v in resources['meta'].items() %}
{{ k }}: {{ v }}
{% endfor -%}
---

{{ super() }}
{%- endblock body %}

{% block codecell -%}
<div class="codecell" markdown="1">
{{ super() }}
</div>
{% endblock codecell %}

{% block input_group -%}
<div class="input_area" markdown="1">
{{ super() }}
</div>
{% endblock input_group %}

{% block output_group -%}
<div class="output_area" markdown="1">
{{ super() }}
</div>
{% endblock output_group %}

