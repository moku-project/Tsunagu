package tsunagu.grpc

import eu.kanade.tachiyomi.animesource.model.AnimeFilter
import eu.kanade.tachiyomi.animesource.model.AnimeFilterList
import sandbox.v1.Sandbox

object AnimeFilterSerde {

    fun AnimeFilterList.toProto(): List<Sandbox.FilterNode> = this.list.map { it.toProtoNode() }

    private fun AnimeFilter<*>.toProtoNode(): Sandbox.FilterNode {
        val builder = Sandbox.FilterNode.newBuilder().setName(name)
        when (this) {
            is AnimeFilter.Header -> builder.setHeader(Sandbox.HeaderFilter.getDefaultInstance())
            is AnimeFilter.Separator -> builder.setSeparator(Sandbox.SeparatorFilter.getDefaultInstance())
            is AnimeFilter.Select<*> -> builder.setSelect(
                Sandbox.SelectFilter.newBuilder()
                    .addAllValues(values.map { it.toString() })
                    .setState(state)
                    .build(),
            )
            is AnimeFilter.Text -> builder.setText(
                Sandbox.TextFilter.newBuilder().setState(state).build(),
            )
            is AnimeFilter.CheckBox -> builder.setCheckbox(
                Sandbox.CheckBoxFilter.newBuilder().setState(state).build(),
            )
            is AnimeFilter.TriState -> builder.setTristate(
                Sandbox.TriStateFilter.newBuilder().setState(state).build(),
            )
            is AnimeFilter.Group<*> -> builder.setGroup(
                Sandbox.GroupFilter.newBuilder()
                    .addAllChildren(
                        state.mapNotNull { (it as? AnimeFilter<*>)?.toProtoNode() },
                    )
                    .build(),
            )
            is AnimeFilter.Sort -> {
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

    fun List<Sandbox.FilterNode>.applyTo(filterList: AnimeFilterList) {
        applyNodes(this, filterList.list)
    }

    private fun applyNodes(nodes: List<Sandbox.FilterNode>, existing: List<AnimeFilter<*>>) {
        val byName = existing.groupBy { it.name }
        for ((i, node) in nodes.withIndex()) {
            val named = byName[node.name]?.singleOrNull()?.takeIf { node.name.isNotEmpty() }
            val target = named ?: existing.getOrNull(i) ?: continue
            applyNodeTo(node, target)
        }
    }

    @Suppress("UNCHECKED_CAST")
    private fun applyNodeTo(node: Sandbox.FilterNode, filter: AnimeFilter<*>) {
        when {
            node.hasSelect() && filter is AnimeFilter.Select<*> ->
                (filter as AnimeFilter<Int>).state = node.select.state
            node.hasText() && filter is AnimeFilter.Text ->
                (filter as AnimeFilter<String>).state = node.text.state
            node.hasCheckbox() && filter is AnimeFilter.CheckBox ->
                (filter as AnimeFilter<Boolean>).state = node.checkbox.state
            node.hasTristate() && filter is AnimeFilter.TriState ->
                (filter as AnimeFilter<Int>).state = node.tristate.state
            node.hasSort() && filter is AnimeFilter.Sort -> {
                val s = node.sort
                (filter as AnimeFilter<AnimeFilter.Sort.Selection?>).state =
                    if (s.hasState) AnimeFilter.Sort.Selection(s.index, s.ascending) else null
            }
            node.hasGroup() && filter is AnimeFilter.Group<*> ->
                applyNodes(node.group.childrenList, filter.state as List<AnimeFilter<*>>)
        }
    }
}
