package tsunagu.grpc

import eu.kanade.tachiyomi.source.model.Filter
import eu.kanade.tachiyomi.source.model.FilterList
import sandbox.v1.Sandbox

object FilterSerde {

    fun FilterList.toProto(): List<Sandbox.FilterNode> = this.list.map { it.toProtoNode() }

    private fun Filter<*>.toProtoNode(): Sandbox.FilterNode {
        val builder = Sandbox.FilterNode.newBuilder().setName(name)
        when (this) {
            is Filter.Header -> builder.setHeader(Sandbox.HeaderFilter.getDefaultInstance())
            is Filter.Separator -> builder.setSeparator(Sandbox.SeparatorFilter.getDefaultInstance())
            is Filter.Select<*> -> builder.setSelect(
                Sandbox.SelectFilter.newBuilder()
                    .addAllValues(values.map { it.toString() })
                    .setState(state)
                    .build(),
            )
            is Filter.Text -> builder.setText(
                Sandbox.TextFilter.newBuilder().setState(state).build(),
            )
            is Filter.CheckBox -> builder.setCheckbox(
                Sandbox.CheckBoxFilter.newBuilder().setState(state).build(),
            )
            is Filter.TriState -> builder.setTristate(
                Sandbox.TriStateFilter.newBuilder().setState(state).build(),
            )
            is Filter.Group<*> -> builder.setGroup(
                Sandbox.GroupFilter.newBuilder()
                    .addAllChildren(
                        state.mapNotNull { (it as? Filter<*>)?.toProtoNode() },
                    )
                    .build(),
            )
            is Filter.Sort -> {
                val sel = state
                val sortBuilder = Sandbox.SortFilter.newBuilder().addAllValues(values.toList())
                if (sel != null) {
                    sortBuilder.setHasState(true).setIndex(sel.index).setAscending(sel.ascending)
                } else {
                    sortBuilder.setHasState(false)
                }
                builder.setSort(sortBuilder.build())
            }
        }
        return builder.build()
    }

    fun List<Sandbox.FilterNode>.applyTo(filterList: FilterList) {
        applyNodes(this, filterList.list)
    }

    private fun applyNodes(nodes: List<Sandbox.FilterNode>, existing: List<Filter<*>>) {
        val byName = existing.groupBy { it.name }
        for ((i, node) in nodes.withIndex()) {
            val named = byName[node.name]?.singleOrNull()?.takeIf { node.name.isNotEmpty() }
            val target = named ?: existing.getOrNull(i) ?: continue
            applyNodeTo(node, target)
        }
    }

    @Suppress("UNCHECKED_CAST")
    private fun applyNodeTo(node: Sandbox.FilterNode, filter: Filter<*>) {
        when {
            node.hasSelect() && filter is Filter.Select<*> ->
                (filter as Filter<Int>).state = node.select.state
            node.hasText() && filter is Filter.Text ->
                (filter as Filter<String>).state = node.text.state
            node.hasCheckbox() && filter is Filter.CheckBox ->
                (filter as Filter<Boolean>).state = node.checkbox.state
            node.hasTristate() && filter is Filter.TriState ->
                (filter as Filter<Int>).state = node.tristate.state
            node.hasSort() && filter is Filter.Sort -> {
                val s = node.sort
                (filter as Filter<Filter.Sort.Selection?>).state =
                    if (s.hasState) Filter.Sort.Selection(s.index, s.ascending) else null
            }
            node.hasGroup() && filter is Filter.Group<*> ->
                applyNodes(node.group.childrenList, filter.state as List<Filter<*>>)

        }
    }
}
